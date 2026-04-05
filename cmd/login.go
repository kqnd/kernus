package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kiev/kernus/internal/auth"
	"github.com/kiev/kernus/internal/config"
	tuiPkg "github.com/kiev/kernus/internal/tui"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Kernus",
	Long: `Authenticate with Kernus. Any username and password are accepted.

If run without flags, launches an interactive TUI login form.

Examples:
  kernus login
  kernus login --username admin --password secret`,
	RunE: func(cmd *cobra.Command, args []string) error {
		email, err := cmd.Flags().GetString("email")
		if err != nil {
			return err
		}
		if email == "" {
			email, err = cmd.Flags().GetString("username")
			if err != nil {
				return err
			}
		}
		password, err := cmd.Flags().GetString("password")
		if err != nil {
			return err
		}
		serverURL, err := cmd.Flags().GetString("server")
		if err != nil {
			return err
		}
		useLocal, err := cmd.Flags().GetBool("local")
		if err != nil {
			return err
		}

		if auth.IsSessionValid() {
			fmt.Println("⚠ You already have an active session.")
			fmt.Print("Do you want to login again? (y/N): ")
			var answer string
			_, err := fmt.Scanln(&answer)
			if err != nil || (answer != "y" && answer != "Y") {
				return nil
			}
		}

		if email == "" || password == "" {
			return tuiPkg.RunLoginApp()
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var client auth.AuthClient
		if useLocal {
			client = auth.NewLocalClient()
		} else {
			serverURL = config.ResolveServerURL(serverURL)
			client = auth.NewHTTPClient(serverURL)
		}

		session, err := client.Login(ctx, email, password)
		if err != nil {
			if errors.Is(err, auth.ErrOAuthOnlyAccount) {
				fmt.Println()
				fmt.Println("✗ This account was created via Google Sign-In and has no password.")
				fmt.Println()
				fmt.Println("  To use the CLI, create an Agent Token from the web dashboard:")
				fmt.Printf("  → %s/dashboard/agent-tokens\n", config.ResolveServerURL(serverURL))
				fmt.Println()
				fmt.Println("  Then configure the agent with:")
				fmt.Println("  kernus token <your-token> --server <server-url> --host <hostname>")
				fmt.Println()
				return nil
			}
			return err
		}

		orgID, _ := auth.ExtractOrgID(session.Token)

		err = auth.SaveSession(&auth.StoredSession{
			Token:     session.Token,
			Username:  session.Username,
			Email:     session.Username,
			UserID:    session.UserID,
			OrgID:     orgID,
			ExpiresAt: session.ExpiresAt,
		})
		if err != nil {
			return err
		}

		fmt.Printf("✓ Logged in as %s\n", session.Username)
		return nil
	},
}

func init() {
	loginCmd.Flags().String("email", "", "Email")
	loginCmd.Flags().String("username", "", "Username")
	loginCmd.Flags().String("password", "", "Password")
	loginCmd.Flags().String("server", "", "Kernus API server URL (precedence: flag > env > config > default)")
	loginCmd.Flags().Bool("local", false, "Use local mock auth client")
	rootCmd.AddCommand(loginCmd)
}
