package cmd

import (
	"fmt"

	"github.com/kern/internal/tui"
	"github.com/spf13/cobra"
)

var group string

var seeCommand = &cobra.Command{
	Use:   "see",
	Short: "View machines monitoring interface",
	Long:  "Launch the TUI interface to monitor machines on the specified server",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Launching monitoring interface...")

		ExitIfIsMissingFields()

		config := &tui.Config{
			Server: "server",
			Group:  group,
		}

		NUNDB_CLIENT.CreateDatabase("kern", "kern-pwd")
		NUNDB_CLIENT.UseDatabase("kern", "kern-pwd")

		app := tui.NewApp(config)
		app.SetNunDBClient(NUNDB_CLIENT)
		if err := app.Run(); err != nil {
			fmt.Printf("error running monitoring interface: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(seeCommand)
	seeCommand.Flags().StringVarP(&group, "group", "g", "", "group")
}
