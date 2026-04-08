package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	enforce    bool
	noAutoAuth bool
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the MCP gateway proxy",
	Long: `Start the Clawkeeper MCP Gateway proxy. In audit mode (default),
the gateway logs all tool calls and flags suspicious activity without
blocking. In enforce mode, tool calls that violate security policies
are blocked.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if enforce {
			fmt.Println("Starting gateway in enforce mode...")
		} else {
			fmt.Println("Starting gateway in audit mode...")
		}
		return nil
	},
}

func init() {
	serverCmd.Flags().BoolVar(&enforce, "enforce", false, "Enable enforce mode (block policy violations)")
	serverCmd.Flags().BoolVar(&noAutoAuth, "no-auto-auth", false, "Disable automatic device authentication")
	rootCmd.AddCommand(serverCmd)
}
