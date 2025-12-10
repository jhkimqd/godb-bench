package metrics

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/pingcap/go-ycsb/pkg/measurement"
)

// FormatMetricsTable captures YCSB output and formats it as a table
func FormatMetricsTable() {
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

	// Parse and format as table
	const tableWidth = 126
	fmt.Println("\n" + strings.Repeat("═", tableWidth))

	// Center the title
	title := "YCSB BENCHMARK RESULTS"
	padding := (tableWidth - len(title)) / 2
	fmt.Println(strings.Repeat(" ", padding) + title)

	fmt.Println(strings.Repeat("═", tableWidth))

	// Table header
	fmt.Printf("│ %-12s │ %10s │ %10s │ %9s │ %9s │ %9s │ %9s │ %9s │ %9s │ %9s │\n",
		"Operation", "Takes(s)", "Count", "OPS", "Avg(µs)", "p50(µs)", "p95(µs)", "p99(µs)", "p99.9(µs)", "Max(µs)")
	fmt.Println(strings.Repeat("─", tableWidth))

	// Parse each line
	scanner := bufio.NewScanner(strings.NewReader(output))
	re := regexp.MustCompile(`^(\S+)\s+-\s+Takes\(s\):\s+([\d.]+),\s+Count:\s+(\d+),\s+OPS:\s+([\d.]+),\s+Avg\(us\):\s+(\d+),\s+Min\(us\):\s+(\d+),\s+Max\(us\):\s+(\d+),\s+50th\(us\):\s+(\d+),\s+90th\(us\):\s+(\d+),\s+95th\(us\):\s+(\d+),\s+99th\(us\):\s+(\d+),\s+99\.9th\(us\):\s+(\d+)`)

	for scanner.Scan() {
		line := scanner.Text()
		matches := re.FindStringSubmatch(line)
		if len(matches) > 0 {
			op := matches[1]
			takes := matches[2]
			count := matches[3]
			ops := matches[4]
			avg := matches[5]
			p50 := matches[8]
			p95 := matches[10]
			p99 := matches[11]
			p999 := matches[12]
			max := matches[7]

			fmt.Printf("│ %-12s │ %10s │ %10s │ %9s │ %9s │ %9s │ %9s │ %9s │ %9s │ %9s │\n",
				op, takes, count, ops, avg, p50, p95, p99, p999, max)
		}
	}

	fmt.Println(strings.Repeat("═", tableWidth))
}
