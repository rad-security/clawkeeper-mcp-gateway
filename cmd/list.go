package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	listHealth bool
	listJSON   bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Show registered MCP servers",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("No servers registered.")
		return nil
	},
}

func init() {
	listCmd.Flags().BoolVar(&listHealth, "health", false, "Include health check status")
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output in JSON format")
	rootCmd.AddCommand(listCmd)
}
