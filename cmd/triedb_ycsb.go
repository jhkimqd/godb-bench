package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/magiconair/properties"
	"github.com/pingcap/go-ycsb/pkg/client"
	"github.com/pingcap/go-ycsb/pkg/measurement"
	"github.com/pingcap/go-ycsb/pkg/prop"
	"github.com/pingcap/go-ycsb/pkg/ycsb"
	"github.com/spf13/cobra"

	_ "github.com/jihwankim/polygon-benchmarks/godb-bench/db"
	"github.com/jihwankim/polygon-benchmarks/godb-bench/metrics"
	_ "github.com/pingcap/go-ycsb/pkg/workload"
)

var (
	triedbWorkloadFile   string
	triedbPropertyFile   string
	triedbPropertyValues []string
)

var triedbYcsbCmd = &cobra.Command{
	Use:   "ycsb",
	Short: "Run the YCSB benchmark on TrieDB",
	Run: func(cmd *cobra.Command, args []string) {
		if triedbWorkloadFile == "" {
			fmt.Println("Please specify a workload file using -w or --workload")
			os.Exit(1)
		}

		props := properties.NewProperties()
		if triedbPropertyFile != "" {
			f, err := os.Open(triedbPropertyFile)
			if err != nil {
				fmt.Printf("Failed to open property file %s: %v\n", triedbPropertyFile, err)
				os.Exit(1)
			}
			defer f.Close()
			data, err := io.ReadAll(f)
			if err != nil {
				fmt.Printf("Failed to read properties from %s: %v\n", triedbPropertyFile, err)
				os.Exit(1)
			}
			if err := props.Load(data, properties.UTF8); err != nil {
				fmt.Printf("Failed to load properties from %s: %v\n", triedbPropertyFile, err)
				os.Exit(1)
			}
		}

		for _, p := range triedbPropertyValues {
			parts := strings.SplitN(p, "=", 2)
			if len(parts) != 2 {
				fmt.Printf("Invalid property format: %s\n", p)
				os.Exit(1)
			}
			props.Set(parts[0], parts[1])
		}

		dbName := "triedb"
		props.Set(prop.DB, dbName)

		// Enable measurement output if not already set
		if props.GetString(prop.MeasurementType, "") == "" {
			props.Set(prop.MeasurementType, "histogram")
		}

		if f, err := os.Open(triedbWorkloadFile); err != nil {
			fmt.Printf("Failed to open workload file %s: %v\n", triedbWorkloadFile, err)
			os.Exit(1)
		} else {
			defer f.Close()
			data, err := io.ReadAll(f)
			if err != nil {
				fmt.Printf("Failed to read workload file %s: %v\n", triedbWorkloadFile, err)
				os.Exit(1)
			}
			p := properties.NewProperties()
			if err := p.Load(data, properties.UTF8); err != nil {
				fmt.Printf("Failed to load properties from workload file %s: %v\n", triedbWorkloadFile, err)
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

		measurement.InitMeasure(props)

		// Wrap DB with measurement wrapper
		tracker := metrics.NewOperationTracker(db)
		wrappedDB := client.DbWrapper{DB: tracker}

		c := client.NewClient(props, wl, wrappedDB)

		fmt.Println("Running workload...")
		c.Run(context.Background())

		fmt.Println("Workload completed. Generating metrics...")
		metrics.FormatMetricsTable(tracker)

		// Generate criterion-style plots
		plotsDir := "./benchmark_plots"
		fmt.Printf("\nGenerating benchmark plots in %s...\n", plotsDir)
		if err := tracker.GeneratePlots(plotsDir); err != nil {
			fmt.Printf("Warning: failed to generate plots: %v\n", err)
		} else {
			fmt.Printf("Plots generated successfully in %s\n", plotsDir)
		}
	},
}
