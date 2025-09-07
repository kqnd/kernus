package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var name string
var sendCommand = &cobra.Command{
	Use:   "send",
	Short: "Send your machine to machines monitoring server",
	Run: func(cmd *cobra.Command, args []string) {
		if name == "" {
			fmt.Println("no name provided")
		} else {
			fmt.Println(name)
		}
	},
}

func init() {
	rootCmd.AddCommand(sendCommand)
	sendCommand.Flags().StringVarP(&name, "name", "n", "", "machine name")
}
