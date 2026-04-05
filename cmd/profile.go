package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/kiev/kernus/internal/auth"
	"github.com/spf13/cobra"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Display current user profile",
	Long: `Show profile information for the currently logged-in user.

Examples:
  kernus profile
  kernus profile --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		session, err := auth.LoadSession()
		if err != nil {
			return fmt.Errorf("not logged in — run 'kernus login' first")
		}

		if !auth.IsSessionValid() {
			return fmt.Errorf("session has expired — run 'kernus login' again")
		}

		jsonOutput, err := cmd.Flags().GetBool("json")
		if err != nil {
			return err
		}

		if jsonOutput {
			data, err := json.MarshalIndent(session, "", "  ")
			if err != nil {
				return fmt.Errorf("cannot marshal profile: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		fmt.Println("┌─────────────────────────────┐")
		fmt.Println("│  User Profile               │")
		fmt.Printf("│  Username : %-16s│\n", session.Username)
		fmt.Printf("│  Expires  : %-16s│\n", session.ExpiresAt.Format("2006-01-02 15:04"))
		fmt.Println("└─────────────────────────────┘")
		return nil
	},
}

func init() {
	profileCmd.Flags().Bool("json", false, "Output profile as JSON")
	rootCmd.AddCommand(profileCmd)
}
