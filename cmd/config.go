package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type JSONConfig struct {
	Server   string `json:"server"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
	Token    string `json:"token"`
}

var server string

var configCommand = &cobra.Command{
	Use:   "config",
	Short: "Set up server for monitoring",
	Run: func(cmd *cobra.Command, args []string) {
		if server == "" {
			fmt.Println("Error: server is required")
			fmt.Println("Usage: kern see --server <server-address>")
			return
		}

		jsonData, err := json.MarshalIndent(JSONConfig{Server: server}, "", "  ")
		if err != nil {
			fmt.Println("occurred an error during config parsing to json: ", err)
			return
		}

		err = os.WriteFile("./config.json", jsonData, 0644)
		if err != nil {
			fmt.Println("occurred an error during saving json config: ", err)
			return
		}

		fmt.Println("Config updated")
	},
}

func init() {
	rootCmd.AddCommand(configCommand)
	configCommand.Flags().StringVarP(&server, "server", "s", "", "nundb server (required)")
}
