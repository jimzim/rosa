package auth

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// NewWhoAmICommand creates the whoami command
func NewWhoAmICommand() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Display current login information",
		Long:  "Display information about the currently logged-in Red Hat account.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWhoAmI(cmd.Context())
		},
	}
}

func runWhoAmI(ctx context.Context) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if cfg.Token == "" {
		return fmt.Errorf("not logged in. Run 'rosa login' first")
	}

	// Create API client
	client, err := api.NewClient(ctx, api.Config{
		URL:   cfg.APIURL,
		Token: cfg.Token,
	})
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Get connection
	conn := client.GetConnection()
	if conn == nil {
		return fmt.Errorf("failed to establish connection")
	}

	// Get current account
	accountResp, err := conn.AccountsMgmt().V1().CurrentAccount().Get().Send()
	if err != nil {
		return fmt.Errorf("failed to get account information: %w", err)
	}

	account := accountResp.Body()

	// Display account information
	output.Info("Current Account:")
	writer := output.NewWriter(output.FormatText)

	accountInfo := map[string]string{
		"Account ID":   account.ID(),
		"Username":     account.Username(),
		"Email":        account.Email(),
		"Name":         fmt.Sprintf("%s %s", account.FirstName(), account.LastName()),
		"Organization": account.Organization().Name(),
	}

	writer.KeyValue(accountInfo)

	// Display quota if available
	// Quota information would be displayed here if available from the API

	// Display environment
	output.Info("\nEnvironment:")
	fmt.Printf("  API URL: %s\n", cfg.APIURL)

	return nil
}
