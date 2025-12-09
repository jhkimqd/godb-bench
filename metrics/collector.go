package metrics

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
)

// Collector collects and tracks benchmark metrics
type Collector struct {
	// Operation counts
	readCount   atomic.Uint64
	updateCount atomic.Uint64
	insertCount atomic.Uint64
	scanCount   atomic.Uint64
	deleteCount atomic.Uint64

	// Latency histograms (in nanoseconds, up to 60 seconds)
	readLatency   *hdrhistogram.Histogram
	updateLatency *hdrhistogram.Histogram
	insertLatency *hdrhistogram.Histogram
	scanLatency   *hdrhistogram.Histogram
	deleteLatency *hdrhistogram.Histogram

	// Amplification metrics (for DBs that support it)
	readAmpCount atomic.Uint64
	readAmpSum   atomic.Uint64

	// Timing
	startTime time.Time
}

// NewCollector creates a new metrics collector
func NewCollector() *Collector {
	return &Collector{
		// Create histograms: min=1ns, max=60s, significant figures=2
		readLatency:   hdrhistogram.New(1, 60000000000, 2),
		updateLatency: hdrhistogram.New(1, 60000000000, 2),
		insertLatency: hdrhistogram.New(1, 60000000000, 2),
		scanLatency:   hdrhistogram.New(1, 60000000000, 2),
		deleteLatency: hdrhistogram.New(1, 60000000000, 2),
		startTime:     time.Now(),
	}
}

// RecordRead records a read operation with its latency
func (c *Collector) RecordRead(latency time.Duration) {
	c.readCount.Add(1)
	_ = c.readLatency.RecordValue(latency.Nanoseconds())
}

// RecordReadWithAmp records a read operation with latency and read amplification
func (c *Collector) RecordReadWithAmp(latency time.Duration, readAmp int) {
	c.RecordRead(latency)
	if readAmp > 0 {
		c.readAmpCount.Add(1)
		c.readAmpSum.Add(uint64(readAmp))
	}
}

// RecordUpdate records an update operation with its latency
func (c *Collector) RecordUpdate(latency time.Duration) {
	c.updateCount.Add(1)
	_ = c.updateLatency.RecordValue(latency.Nanoseconds())
}

// RecordInsert records an insert operation with its latency
func (c *Collector) RecordInsert(latency time.Duration) {
	c.insertCount.Add(1)
	_ = c.insertLatency.RecordValue(latency.Nanoseconds())
}

// RecordScan records a scan operation with its latency
func (c *Collector) RecordScan(latency time.Duration) {
	c.scanCount.Add(1)
	_ = c.scanLatency.RecordValue(latency.Nanoseconds())
}

// RecordDelete records a delete operation with its latency
func (c *Collector) RecordDelete(latency time.Duration) {
	c.deleteCount.Add(1)
	_ = c.deleteLatency.RecordValue(latency.Nanoseconds())
}

// PrintProgress prints current progress (called periodically during benchmark)
func (c *Collector) PrintProgress(opsCompleted int) {
	elapsed := time.Since(c.startTime)
	throughput := float64(opsCompleted) / elapsed.Seconds()
	fmt.Printf("Progress: %d ops, %.1f ops/sec\n", opsCompleted, throughput)
}

// PrintSummary prints a comprehensive summary of all metrics
func (c *Collector) PrintSummary(dbMetrics interface{}) {
	elapsed := time.Since(c.startTime)

	fmt.Println("\n____optype__elapsed_____ops(total)___ops/sec(cum)__avg(ms)__p50(ms)__p95(ms)__p99(ms)_pMax(ms)")

	c.printOpSummary("read", c.readCount.Load(), c.readLatency, elapsed)
	c.printOpSummary("update", c.updateCount.Load(), c.updateLatency, elapsed)
	c.printOpSummary("insert", c.insertCount.Load(), c.insertLatency, elapsed)
	c.printOpSummary("scan", c.scanCount.Load(), c.scanLatency, elapsed)
	c.printOpSummary("delete", c.deleteCount.Load(), c.deleteLatency, elapsed)

	// Print overall summary
	fmt.Println()
	totalOps := c.readCount.Load() + c.updateCount.Load() + c.insertCount.Load() + c.scanCount.Load() + c.deleteCount.Load()

	readAmpCount := c.readAmpCount.Load()
	readAmpSum := c.readAmpSum.Load()
	avgReadAmp := 0.0
	if readAmpCount > 0 {
		avgReadAmp = float64(readAmpSum) / float64(readAmpCount)
	}

	fmt.Printf("Benchmark Summary:\n")
	fmt.Printf("  Total operations: %d\n", totalOps)
	fmt.Printf("  Total elapsed: %.1fs\n", elapsed.Seconds())
	fmt.Printf("  Throughput: %.1f ops/sec\n", float64(totalOps)/elapsed.Seconds())
	if readAmpCount > 0 {
		fmt.Printf("  Avg Read Amplification: %.2f\n", avgReadAmp)
	}

	// Print DB-specific metrics if available
	c.printDBMetrics(dbMetrics)
}

// printOpSummary prints a single operation type summary line
func (c *Collector) printOpSummary(name string, count uint64, hist *hdrhistogram.Histogram, elapsed time.Duration) {
	if count == 0 {
		return
	}

	fmt.Printf("%10s %7.1fs %14d %14.1f %8.1f %8.1f %8.1f %8.1f %8.1f\n",
		name,
		elapsed.Seconds(),
		count,
		float64(count)/elapsed.Seconds(),
		float64(hist.Mean())/1e6,
		float64(hist.ValueAtQuantile(50))/1e6,
		float64(hist.ValueAtQuantile(95))/1e6,
		float64(hist.ValueAtQuantile(99))/1e6,
		float64(hist.Max())/1e6,
	)
}

// printDBMetrics prints database-specific metrics
func (c *Collector) printDBMetrics(dbMetrics interface{}) {
	// For now, we'll add a placeholder for DB-specific metrics
	// This can be extended based on what PebbleDB and TrieDB expose
	if dbMetrics == nil {
		return
	}

	fmt.Println("\nDatabase-specific metrics:")
	// Type switch for different DB metrics
	switch m := dbMetrics.(type) {
	case string:
		// If metrics are provided as a string (e.g., from Pebble's Metrics().String())
		fmt.Println(m)
	default:
		// For PebbleDB Metrics, we can try to extract key information
		// We'll use reflection or type assertion for common metrics types
		fmt.Printf("  Raw metrics available (type: %T)\n", dbMetrics)

		// Try to print as a stringer
		if s, ok := dbMetrics.(fmt.Stringer); ok {
			fmt.Println(s.String())
		}
	}
}
