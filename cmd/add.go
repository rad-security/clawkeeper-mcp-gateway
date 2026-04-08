package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	addEnv     string
	addHeaders []string
)

var addCmd = &cobra.Command{
	Use:   "add <name> <command_or_url>",
	Short: "Register an MCP server with the gateway",
	Long: `Register an MCP server to be proxied through the gateway. The server
can be a local stdio command or a remote SSE URL.

Examples:
  clawkeeper-mcp-gateway add filesystem npx -y @modelcontextprotocol/server-filesystem /tmp
  clawkeeper-mcp-gateway add remote-api https://api.example.com/mcp/sse --header "Authorization:Bearer tok"`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		fmt.Printf("Added server: %s\n", name)
		return nil
	},
}

func init() {
	addCmd.Flags().StringVar(&addEnv, "env", "", "JSON environment variables for the server process")
	addCmd.Flags().StringSliceVar(&addHeaders, "header", nil, "Headers for SSE connections (key:value)")
	rootCmd.AddCommand(addCmd)
}
