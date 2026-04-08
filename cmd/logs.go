package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	logsFollow bool
	logsLines  int
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View gateway event logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		if logsFollow {
			fmt.Println("Following gateway logs...")
		} else {
			fmt.Printf("Showing last %d log entries.\n", logsLines)
		}
		return nil
	},
}

func init() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
	logsCmd.Flags().IntVarP(&logsLines, "lines", "l", 20, "Number of lines to show")
	rootCmd.AddCommand(logsCmd)
}
