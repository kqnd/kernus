package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	nundb "github.com/viewfromaside/nun-db-go"
)

var NUNDB_CLIENT *nundb.Client
var CONFIG *JSONConfig

var rootCmd = &cobra.Command{
	Use:   "kern",
	Short: "CLI/TUI machine monitoring app",
}

func init() {
	config := &JSONConfig{}
	ReadConfigJSONFile(config)
	CONFIG = config

	if config.Server != "" {
		client, _ := nundb.NewClient(config.Server, config.Username, config.Password)
		NUNDB_CLIENT = client
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
