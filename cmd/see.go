package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var server, group string

var seeCommand = &cobra.Command{
	Use:   "see",
	Short: "View machines monitoring interface",
	Long:  "Launch the TUI interface to monitor machines on the specified server",
	Run: func(cmd *cobra.Command, args []string) {
		if server == "" {
			fmt.Println("Error: server is required")
			fmt.Println("Usage: kern see --server <server-address>")
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(seeCommand)
	seeCommand.Flags().StringVarP(&server, "server", "s", "", "nundb server address (required)")
}
