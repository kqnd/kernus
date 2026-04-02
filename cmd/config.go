package cmd

import (
	"fmt"

	"github.com/kiev/kernus/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure Kernus settings",
	Long: `Set Kernus settings such as credentials and API server.

Examples:
  kernus config --username user@example.com --password secret
  kernus config --server https://api.kernus.app`,
	RunE: func(cmd *cobra.Command, args []string) error {
		username, err := cmd.Flags().GetString("username")
		if err != nil {
			return err
		}
		password, err := cmd.Flags().GetString("password")
		if err != nil {
			return err
		}
		server, err := cmd.Flags().GetString("server")
		if err != nil {
			return err
		}

		if !cmd.Flags().Changed("username") && !cmd.Flags().Changed("password") && !cmd.Flags().Changed("server") {
			return fmt.Errorf("nothing to update — provide at least one flag")
		}

		existing, loadErr := config.Load()
		if loadErr != nil {
			existing = &config.Config{}
		}

		c := &config.Config{
			Server:   existing.Server,
			Username: existing.Username,
			Password: existing.Password,
			Database: existing.Database,
			Token:    existing.Token,
		}

		if cmd.Flags().Changed("server") {
			c.Server = server
		}
		if cmd.Flags().Changed("username") {
			c.Username = username
		}
		if cmd.Flags().Changed("password") {
			c.Password = password
		}

		path, err := config.Save(c)
		if err != nil {
			return err
		}

		fmt.Printf("✓ Configuration saved to %s\n", path)
		if c.Server != "" {
			fmt.Printf("✓ Default server: %s\n", c.Server)
		}
		return nil
	},
}

func init() {
	configCmd.Flags().String("username", "", "Username")
	configCmd.Flags().String("password", "", "Password")
	configCmd.Flags().String("server", "", "Default Kernus API server URL")

	rootCmd.AddCommand(configCmd)
}
