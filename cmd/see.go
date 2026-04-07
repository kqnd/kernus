package cmd

import (
	tuiPkg "github.com/kiev/kernus/internal/tui"
	"github.com/spf13/cobra"
)

var seeCmd = &cobra.Command{
	Use:   "see",
	Short: "Launch the monitoring TUI",
	Long: `Start the interactive terminal UI for local Docker monitoring.

Examples:
  kernus see
  kernus see --machines
  kernus see --refresh 5
  kernus see --docker-host tcp://192.168.1.10:2375`,
	RunE: func(cmd *cobra.Command, args []string) error {
		group, err := cmd.Flags().GetString("group")
		if err != nil {
			return err
		}
		dockerHost, err := cmd.Flags().GetString("docker-host")
		if err != nil {
			return err
		}
		refresh, err := cmd.Flags().GetInt("refresh")
		if err != nil {
			return err
		}
		machines, err := cmd.Flags().GetBool("machines")
		if err != nil {
			return err
		}
		useMock, err := cmd.Flags().GetBool("mock")
		if err != nil {
			return err
		}

		tuiCfg := tuiPkg.Config{
			Group:        group,
			RefreshRate:  refresh,
			DockerHost:   dockerHost,
			ShowMachines: machines,
			Session:      nil,
			UseMock:      useMock,
		}

		return tuiPkg.Run(tuiCfg)
	},
}

func init() {
	seeCmd.Flags().String("group", "", "Filter machines by group")
	seeCmd.Flags().String("docker-host", "", "Docker daemon URL (default: local socket)")
	seeCmd.Flags().Int("refresh", 3, "Refresh interval in seconds")
	seeCmd.Flags().Bool("machines", false, "Show remote machines panel instead of containers")
	seeCmd.Flags().Bool("mock", false, "Use mock data instead of Docker daemon")
	rootCmd.AddCommand(seeCmd)
}
