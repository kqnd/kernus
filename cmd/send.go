package cmd

import "github.com/spf13/cobra"

var name string

var sendCommand = &cobra.Command{
	Use:   "send",
	Short: "Send machine to monitoring server",
	Long:  "Send your machine to the monitoring server using the CLI options",
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func init() {
	rootCmd.AddCommand(sendCommand)
	sendCommand.Flags().StringVarP(&name, "name", "n", "", "machine name (required)")
}
