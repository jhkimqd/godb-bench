package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var triedbCmd = &cobra.Command{
	Use:   "triedb",
	Short: "Benchmark TrieDB",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Use 'triedb [command]' to run a specific benchmark.")
	},
}
