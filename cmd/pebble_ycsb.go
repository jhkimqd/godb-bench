package cmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/magiconair/properties"
	"github.com/pingcap/go-ycsb/pkg/client"
	"github.com/pingcap/go-ycsb/pkg/measurement"
	"github.com/pingcap/go-ycsb/pkg/prop"
	"github.com/pingcap/go-ycsb/pkg/ycsb"
	"github.com/spf13/cobra"

	_ "github.com/jihwankim/polygon-benchmarks/godb-bench/db"
	_ "github.com/pingcap/go-ycsb/pkg/workload"
)

var (
	propertyFile   string
	propertyValues []string
	workloadFile   string
)

// formatMetricsTable captures YCSB output and formats it as a table
func formatMetricsTable() {
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
	fmt.Println("\n" + strings.Repeat("═", 132))
	fmt.Println("                                    YCSB BENCHMARK RESULTS")
	fmt.Println(strings.Repeat("═", 132))

	// Table header
	fmt.Printf("│ %-12s │ %10s │ %10s │ %9s │ %9s │ %9s │ %9s │ %9s │ %9s │ %9s │\n",
		"Operation", "Takes(s)", "Count", "OPS", "Avg(µs)", "p50(µs)", "p95(µs)", "p99(µs)", "p99.9(µs)", "Max(µs)")
	fmt.Println(strings.Repeat("─", 132))

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

	fmt.Println(strings.Repeat("═", 132))
}

var ycsbCmd = &cobra.Command{
	Use:   "ycsb",
	Short: "Run the YCSB benchmark on PebbleDB",
	Run: func(cmd *cobra.Command, args []string) {
		if workloadFile == "" {
			fmt.Println("Please specify a workload file using -w or --workload")
			os.Exit(1)
		}

		props := properties.NewProperties()
		// Load properties from file
		if propertyFile != "" {
			f, err := os.Open(propertyFile)
			if err != nil {
				fmt.Printf("Failed to open property file %s: %v\n", propertyFile, err)
				os.Exit(1)
			}
			defer f.Close()
			data, err := io.ReadAll(f)
			if err != nil {
				fmt.Printf("Failed to read properties from %s: %v\n", propertyFile, err)
				os.Exit(1)
			}
			if err := props.Load(data, properties.UTF8); err != nil {
				fmt.Printf("Failed to load properties from %s: %v\n", propertyFile, err)
				os.Exit(1)
			}
		}

		// Load properties from command line
		for _, p := range propertyValues {
			parts := strings.SplitN(p, "=", 2)
			if len(parts) != 2 {
				fmt.Printf("Invalid property format: %s\n", p)
				os.Exit(1)
			}
			props.Set(parts[0], parts[1])
		}

		dbName := "pebble"
		props.Set(prop.DB, dbName)

		// Enable measurement output if not already set
		if props.GetString(prop.MeasurementType, "") == "" {
			props.Set(prop.MeasurementType, "histogram")
		}

		// Make sure we do transactions (not just load)
		props.Set(prop.DoTransactions, "true")

		// The workload file should be loaded as a property file.
		// See https://github.com/pingcap/go-ycsb/blob/master/cmd/go-ycsb/main.go
		if f, err := os.Open(workloadFile); err != nil {
			fmt.Printf("Failed to open workload file %s: %v\n", workloadFile, err)
			os.Exit(1)
		} else {
			defer f.Close()
			data, err := io.ReadAll(f)
			if err != nil {
				fmt.Printf("Failed to read workload file %s: %v\n", workloadFile, err)
				os.Exit(1)
			}
			p := properties.NewProperties()
			if err := p.Load(data, properties.UTF8); err != nil {
				fmt.Printf("Failed to load properties from workload file %s: %v\n", workloadFile, err)
				os.Exit(1)
			} else {
				props.Merge(p)
			}
		}

		workloadName := props.GetString(prop.Workload, "core")
		workloadCreator := ycsb.GetWorkloadCreator(workloadName)
		wl, err := workloadCreator.Create(props)
		if err != nil {
			fmt.Printf("Failed to create workload: %v\n", err)
			os.Exit(1)
		}

		dbCreator := ycsb.GetDBCreator(dbName)
		if dbCreator == nil {
			fmt.Printf("DB creator for %s not found\n", dbName)
			os.Exit(1)
		}

		db, err := dbCreator.Create(props)
		if err != nil {
			fmt.Printf("Failed to create DB: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		// Initialize YCSB measurement system
		measurement.InitMeasure(props)

		// Wrap DB with measurement wrapper
		wrappedDB := client.DbWrapper{DB: db}

		c := client.NewClient(props, wl, wrappedDB)

		fmt.Println("Running workload...")
		c.Run(context.Background())

		fmt.Println("Workload completed. Generating metrics...")

		// Print YCSB metrics in table format
		formatMetricsTable()

		// Print PebbleDB-specific metrics if available
		type pebbleMetricsProvider interface {
			Metrics() interface{}
		}
		if pdb, ok := db.(pebbleMetricsProvider); ok {
			fmt.Println("\n" + strings.Repeat("=", 80))
			fmt.Println("PebbleDB Metrics:")
			fmt.Println(strings.Repeat("=", 80))
			if metrics := pdb.Metrics(); metrics != nil {
				if s, ok := metrics.(fmt.Stringer); ok {
					fmt.Println(s.String())
				}
			}
		}
	},
}
