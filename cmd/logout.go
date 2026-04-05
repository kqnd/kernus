package cmd

import (
	"fmt"

	"github.com/kiev/kernus/internal/auth"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from Kernus",
	Long: `End your current session.

Examples:
  kernus logout`,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := auth.LoadSession()
		if err != nil {
			return fmt.Errorf("not logged in — nothing to logout from")
		}

		err = auth.DeleteSession()
		if err != nil {
			return fmt.Errorf("failed to delete local session: %w", err)
		}

		fmt.Println("✓ Logged out successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
