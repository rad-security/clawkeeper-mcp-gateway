package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	verbose    bool
	configPath string
)

var rootCmd = &cobra.Command{
	Use:   "clawkeeper-mcp-gateway",
	Short: "MCP gateway with threat detection and policy enforcement",
	Long: `Clawkeeper MCP Gateway is an open-source security proxy for the
Model Context Protocol (MCP). It sits between AI clients and MCP
servers, inspecting every tool call and response for threats.

Modes:
  audit    Log suspicious activity without blocking (default)
  enforce  Block tool calls that violate security policies

The gateway supports stdio-based MCP servers and provides real-time
threat detection, sensitive data scanning, and policy enforcement.`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to config file")
}
