package cmd

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/kiev/kernus/internal/auth"
	"github.com/kiev/kernus/internal/metrics"
	"github.com/spf13/cobra"
)

var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send machine metrics to Kernus",
	Long: `Collect and send local machine metrics (CPU, RAM, disk) to the server.

Examples:
  kernus send --name "my-server" --group "backend"
  kernus send --name "db-01" --group "database" --interval 10`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := cmd.Flags().GetString("name")
		if err != nil {
			return err
		}
		group, err := cmd.Flags().GetString("group")
		if err != nil {
			return err
		}
		interval, err := cmd.Flags().GetInt("interval")
		if err != nil {
			return err
		}

		_, err = auth.LoadSession()
		if err != nil {
			return fmt.Errorf("not logged in — run 'kernus login' first")
		}

		if !auth.IsSessionValid() {
			return fmt.Errorf("session has expired — run 'kernus login' again")
		}

		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		collector := metrics.NewCollector()

		fmt.Printf("Sending metrics as '%s' (group: %s) every %ds\n", name, group, interval)
		fmt.Println("Press Ctrl+C to stop")
		fmt.Println()

		ticker := time.NewTicker(time.Duration(interval) * time.Second)
		defer ticker.Stop()

		sendSnapshot := func() {
			snapshot, err := collector.Collect()
			if err != nil {
				fmt.Printf("✗ %s | Error: %v\n", time.Now().Format("15:04:05"), err)
				return
			}
			fmt.Printf("✓ %s | CPU: %.1f%% | MEM: %s/%s | DISK: %s/%s\n",
				snapshot.CollectedAt.Format("15:04:05"),
				snapshot.CPUPercent,
				formatBytes(snapshot.MemoryUsed),
				formatBytes(snapshot.MemoryTotal),
				formatBytes(snapshot.DiskUsed),
				formatBytes(snapshot.DiskTotal),
			)
		}

		sendSnapshot()

		for {
			select {
			case <-ctx.Done():
				fmt.Println("\n✓ Metrics collection stopped")
				return nil
			case <-ticker.C:
				sendSnapshot()
			}
		}
	},
}

func formatBytes(b int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1fGB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1fMB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1fKB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%dB", b)
	}
}

func init() {
	sendCmd.Flags().String("name", "", "Machine name (required)")
	sendCmd.Flags().String("group", "default", "Machine group")
	sendCmd.Flags().Int("interval", 5, "Send interval in seconds")

	err := sendCmd.MarkFlagRequired("name")
	if err != nil {
		panic(err)
	}

	rootCmd.AddCommand(sendCmd)
}
