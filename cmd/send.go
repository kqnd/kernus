package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var name string

var sendCommand = &cobra.Command{
	Use:   "send",
	Short: "Send machine to monitoring server",
	Long:  "Send your machine to the monitoring server using the CLI options",
	Run: func(cmd *cobra.Command, args []string) {

		// ExitIfIsMissingFields()

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		fmt.Println("starting sending process... (ctrl + c for stop)")

		for {
			select {
			case <-ctx.Done():
				fmt.Println("received interrupt signal")
				return
			default:
				fmt.Println("sending metrics")
				time.Sleep(2 * time.Second)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(sendCommand)
	sendCommand.Flags().StringVarP(&name, "name", "n", "", "machine name (required)")
}
