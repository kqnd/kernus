package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/kiev/kernus/internal/config"
	"github.com/spf13/cobra"
)

var cfg *config.Config
var currentVersion = "dev"

var rootCmd = &cobra.Command{
	Use:   "kernus",
	Short: "Kernus — Infrastructure Monitor CLI/TUI",
	Long: `Kernus is a terminal-based infrastructure monitoring tool.

It allows you to:
  - Monitor Docker containers in real time
  - Monitor remote machine metrics (CPU, RAM, disk)
  - Control containers via keyboard shortcuts

Examples:
  kernus config --username admin --password secret
  kernus login
	kernus token create "prod-server-01"
  kernus token kn_live_xxx --server https://api.kernus.app --host prod-server-01
	kernus agent start
	kernus agent stop
  kernus see
  kernus see --machines
  kernus send --name "my-server" --group "backend"`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "config" || cmd.Name() == "help" || cmd.Name() == "login" || cmd.Name() == "see" || cmd.Name() == "stop" {
			return nil
		}
		loaded, err := config.Load()
		if err != nil {
			cfg = &config.Config{}
			return nil
		}
		cfg = loaded
		return nil
	},
}

func SetVersion(version string) {
	version = strings.TrimSpace(version)
	if version == "" {
		version = "dev"
	}
	currentVersion = version
	rootCmd.Version = version
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
