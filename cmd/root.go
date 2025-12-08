package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "godb-bench",
	Short: "A benchmark tool for PebbleDB and TrieDB",
	Long:  `A CLI tool to run benchmarks on different key-value stores.`,
}

func Execute() {
	initCommands()
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initCommands() {
	// Add pebble command and its subcommands
	RootCmd.AddCommand(pebbleCmd)
	pebbleCmd.AddCommand(ycsbCmd)
	ycsbCmd.Flags().StringVarP(&workloadFile, "workload", "w", "", "Path to the YCSB workload file")
	ycsbCmd.Flags().StringVarP(&propertyFile, "property_file", "P", "", "Path to the YCSB property file")
	ycsbCmd.Flags().StringArrayVarP(&propertyValues, "prop", "p", nil, "YCSB property (e.g. -p key=value)")

	// Add triedb command and its subcommands
	RootCmd.AddCommand(triedbCmd)
	triedbCmd.AddCommand(triedbBenchCmd)
	triedbCmd.AddCommand(triedbYcsbCmd)
	triedbYcsbCmd.Flags().StringVarP(&triedbWorkloadFile, "workload", "w", "", "Path to the YCSB workload file")
	triedbYcsbCmd.Flags().StringVarP(&triedbPropertyFile, "property_file", "P", "", "Path to the YCSB property file")
	triedbYcsbCmd.Flags().StringArrayVarP(&triedbPropertyValues, "prop", "p", nil, "YCSB property (e.g. -p key=value)")
}
