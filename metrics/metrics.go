package metrics

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/pingcap/go-ycsb/pkg/measurement"
	"github.com/pingcap/go-ycsb/pkg/ycsb"
)

type OperationTracker struct {
	ycsb.DB
	mu      sync.Mutex
	timings map[string]*OperationTiming
	plots   *BenchmarkPlots
}

type OperationTiming struct {
	Count     int64
	TotalTime time.Duration
	StartTime time.Time
}

func NewOperationTracker(db ycsb.DB) *OperationTracker {
	return &OperationTracker{
		DB:      db,
		timings: make(map[string]*OperationTiming),
		plots:   NewBenchmarkPlots(),
	}
}

func (ot *OperationTracker) track(op string, start time.Time) {
	elapsed := time.Since(start)

	ot.mu.Lock()
	defer ot.mu.Unlock()

	if _, exists := ot.timings[op]; !exists {
		ot.timings[op] = &OperationTiming{StartTime: start}
	}

	ot.timings[op].Count++
	ot.timings[op].TotalTime += elapsed

	// Record sample for plotting (sample index auto-increments)
	ot.plots.AddSample(op, elapsed)
}

func (ot *OperationTracker) Insert(ctx context.Context, table string, key string, values map[string][]byte) error {
	start := time.Now()
	err := ot.DB.Insert(ctx, table, key, values)
	ot.track("INSERT", start)
	return err
}

func (ot *OperationTracker) Update(ctx context.Context, table string, key string, values map[string][]byte) error {
	start := time.Now()
	err := ot.DB.Update(ctx, table, key, values)
	ot.track("UPDATE", start)
	return err
}

func (ot *OperationTracker) Read(ctx context.Context, table string, key string, fields []string) (map[string][]byte, error) {
	start := time.Now()
	result, err := ot.DB.Read(ctx, table, key, fields)
	ot.track("READ", start)
	return result, err
}

func (ot *OperationTracker) Scan(ctx context.Context, table string, startKey string, count int, fields []string) ([]map[string][]byte, error) {
	start := time.Now()
	result, err := ot.DB.Scan(ctx, table, startKey, count, fields)
	ot.track("SCAN", start)
	return result, err
}

func (ot *OperationTracker) Delete(ctx context.Context, table string, key string) error {
	start := time.Now()
	err := ot.DB.Delete(ctx, table, key)
	ot.track("DELETE", start)
	return err
}

// FormatMetricsTable captures YCSB output and formats it as a table
func FormatMetricsTable(tracker *OperationTracker) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Get the output
	measurement.Output()

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Get timing data from tracker
	tracker.mu.Lock()
	timingData := make(map[string]*OperationTiming)
	for op, timing := range tracker.timings {
		timingData[op] = timing
	}
	tracker.mu.Unlock()

	// Parse and format as table
	const tableWidth = 126
	fmt.Println("\n" + strings.Repeat("═", tableWidth))

	// Center the title
	title := "YCSB BENCHMARK RESULTS"
	padding := (tableWidth - len(title)) / 2
	fmt.Println(strings.Repeat(" ", padding) + title)

	fmt.Println(strings.Repeat("═", tableWidth))

	// Table header - replaced Takes(s) with Total(ms)
	fmt.Printf("│ %-12s │ %10s │ %10s │ %9s │ %9s │ %9s │ %9s │ %9s │ %9s │ %9s │\n",
		"Operation", "Total(ms)", "Count", "OPS", "Avg(µs)", "p50(µs)", "p95(µs)", "p99(µs)", "p99.9(µs)", "Max(µs)")
	fmt.Println(strings.Repeat("─", tableWidth))

	// Parse each line
	scanner := bufio.NewScanner(strings.NewReader(output))
	re := regexp.MustCompile(`^(\S+)\s+-\s+Takes\(s\):\s+([\d.]+),\s+Count:\s+(\d+),\s+OPS:\s+([\d.]+),\s+Avg\(us\):\s+(\d+),\s+Min\(us\):\s+(\d+),\s+Max\(us\):\s+(\d+),\s+50th\(us\):\s+(\d+),\s+90th\(us\):\s+(\d+),\s+95th\(us\):\s+(\d+),\s+99th\(us\):\s+(\d+),\s+99\.9th\(us\):\s+(\d+)`)

	// Store rows to print, with TOTAL row separate
	var rows []string
	var totalRow string

	for scanner.Scan() {
		line := scanner.Text()
		matches := re.FindStringSubmatch(line)
		if len(matches) > 0 {
			op := matches[1]
			count := matches[3]
			ops := matches[4]
			avg := matches[5]
			p50 := matches[8]
			p95 := matches[10]
			p99 := matches[11]
			p999 := matches[12]
			max := matches[7]

			// Get actual timing from tracker with higher precision
			totalMs := "N/A"
			if op == "TOTAL" {
				// Sum up all operation times for TOTAL row
				var totalTime time.Duration
				for _, timing := range timingData {
					totalTime += timing.TotalTime
				}
				totalMs = fmt.Sprintf("%.3f", float64(totalTime.Microseconds())/1000.0)
			} else if timing, exists := timingData[op]; exists {
				totalMs = fmt.Sprintf("%.3f", float64(timing.TotalTime.Microseconds())/1000.0)
			}

			rowStr := fmt.Sprintf("│ %-12s │ %10s │ %10s │ %9s │ %9s │ %9s │ %9s │ %9s │ %9s │ %9s │\n",
				op, totalMs, count, ops, avg, p50, p95, p99, p999, max)

			// Separate TOTAL row
			if op == "TOTAL" {
				totalRow = rowStr
			} else {
				rows = append(rows, rowStr)
			}
		}
	}

	// Print all rows except TOTAL
	for _, row := range rows {
		fmt.Print(row)
	}

	// Print TOTAL row last
	if totalRow != "" {
		fmt.Print(totalRow)
	}

	fmt.Println(strings.Repeat("═", tableWidth))
}

// GeneratePlots creates criterion-style scatter plots for the tracked operations
func (ot *OperationTracker) GeneratePlots(outputDir string) error {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	if err := ot.plots.GeneratePlots(outputDir); err != nil {
		return fmt.Errorf("failed to generate plots: %w", err)
	}

	return nil
}

// PrintStatistics prints criterion-style additional statistics
func (ot *OperationTracker) PrintStatistics() {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	ot.plots.PrintStatistics()
}
