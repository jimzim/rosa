// Package interactive provides interactive prompts and forms for the CLI
package interactive

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// Theme returns the default theme for interactive prompts
func Theme() *huh.Theme {
	t := huh.ThemeCharm()
	t.Focused.Title = t.Focused.Title.Foreground(lipgloss.Color("205"))
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(lipgloss.Color("205"))
	return t
}

// PromptString prompts for a string value
func PromptString(title, description string, defaultValue string, required bool) (string, error) {
	var value string

	input := huh.NewInput().
		Title(title).
		Value(&value)

	if description != "" {
		input.Description(description)
	}

	if defaultValue != "" {
		input.Placeholder(defaultValue)
	}

	if required {
		input.Validate(func(s string) error {
			if strings.TrimSpace(s) == "" && defaultValue == "" {
				return fmt.Errorf("value is required")
			}
			return nil
		})
	}

	form := huh.NewForm(huh.NewGroup(input)).WithTheme(Theme())

	if err := form.Run(); err != nil {
		return "", err
	}

	if value == "" && defaultValue != "" {
		value = defaultValue
	}

	return value, nil
}

// PromptBool prompts for a boolean value
func PromptBool(title, description string, defaultValue bool) (bool, error) {
	var value bool = defaultValue

	confirm := huh.NewConfirm().
		Title(title).
		Value(&value).
		Affirmative("Yes").
		Negative("No")

	if description != "" {
		confirm.Description(description)
	}

	form := huh.NewForm(huh.NewGroup(confirm)).WithTheme(Theme())

	if err := form.Run(); err != nil {
		return false, err
	}

	return value, nil
}

// PromptSelect prompts for a selection from a list
func PromptSelect[T comparable](title, description string, options []SelectOption[T], defaultValue T) (T, error) {
	var value T = defaultValue

	huhOptions := make([]huh.Option[T], 0, len(options))
	for _, opt := range options {
		huhOptions = append(huhOptions, huh.NewOption(opt.Label, opt.Value))
	}

	sel := huh.NewSelect[T]().
		Title(title).
		Options(huhOptions...).
		Value(&value)

	if description != "" {
		sel.Description(description)
	}

	form := huh.NewForm(huh.NewGroup(sel)).WithTheme(Theme())

	if err := form.Run(); err != nil {
		var zero T
		return zero, err
	}

	return value, nil
}

// SelectOption represents an option for selection
type SelectOption[T comparable] struct {
	Label string
	Value T
}

// PromptMultiSelect prompts for multiple selections from a list
func PromptMultiSelect[T comparable](title, description string, options []SelectOption[T], defaultValues []T) ([]T, error) {
	values := defaultValues

	huhOptions := make([]huh.Option[T], 0, len(options))
	for _, opt := range options {
		huhOptions = append(huhOptions, huh.NewOption(opt.Label, opt.Value))
	}

	multiSelect := huh.NewMultiSelect[T]().
		Title(title).
		Options(huhOptions...).
		Value(&values)

	if description != "" {
		multiSelect.Description(description)
	}

	// Set default selected items
	if len(defaultValues) > 0 {
		multiSelect.Value(&defaultValues)
	}

	form := huh.NewForm(huh.NewGroup(multiSelect)).WithTheme(Theme())

	if err := form.Run(); err != nil {
		return nil, err
	}

	return values, nil
}

// ClusterCreateForm represents an interactive form for cluster creation
type ClusterCreateForm struct {
	Name           string
	Region         string
	Version        string
	MultiAZ        bool
	PrivateLink    bool
	BillingAccount string
}

// PromptClusterCreate shows an interactive form for cluster creation
func PromptClusterCreate(defaults ClusterCreateForm) (*ClusterCreateForm, error) {
	form := defaults

	// Create a multi-step form
	huhForm := huh.NewForm(
		// Step 1: Basic Information
		huh.NewGroup(
			huh.NewInput().
				Title("Cluster Name").
				Description("A unique name for your cluster").
				Value(&form.Name).
				Validate(ValidateClusterName),

			huh.NewSelect[string]().
				Title("AWS Region").
				Description("The AWS region where the cluster will be created").
				Options(
					huh.NewOption("US East (N. Virginia)", "us-east-1"),
					huh.NewOption("US East (Ohio)", "us-east-2"),
					huh.NewOption("US West (Oregon)", "us-west-2"),
					huh.NewOption("EU (Ireland)", "eu-west-1"),
					huh.NewOption("EU (Frankfurt)", "eu-central-1"),
					huh.NewOption("Asia Pacific (Singapore)", "ap-southeast-1"),
					huh.NewOption("Asia Pacific (Sydney)", "ap-southeast-2"),
				).
				Value(&form.Region),
		).Title("Basic Configuration"),

		// Step 2: Advanced Options
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("OpenShift Version").
				Description("The version of OpenShift to install").
				Options(
					huh.NewOption("4.14.0 (Stable)", "4.14.0"),
					huh.NewOption("4.13.19 (Stable)", "4.13.19"),
					huh.NewOption("4.15.0-rc.1 (Preview)", "4.15.0-rc.1"),
				).
				Value(&form.Version),

			huh.NewConfirm().
				Title("Multi-AZ Deployment").
				Description("Deploy control plane across multiple availability zones for high availability").
				Value(&form.MultiAZ).
				Affirmative("Yes").
				Negative("No"),

			huh.NewConfirm().
				Title("PrivateLink").
				Description("Use AWS PrivateLink for private connectivity to the cluster").
				Value(&form.PrivateLink).
				Affirmative("Yes").
				Negative("No"),
		).Title("Advanced Configuration"),

		// Step 3: Billing (optional)
		huh.NewGroup(
			huh.NewInput().
				Title("Billing Account ID (Optional)").
				Description("AWS account ID for billing (leave empty to use current account)").
				Value(&form.BillingAccount).
				Validate(ValidateBillingAccount),
		).Title("Billing Configuration"),
	).WithTheme(Theme())

	if err := huhForm.Run(); err != nil {
		return nil, err
	}

	return &form, nil
}

// ValidateClusterName validates a cluster name
func ValidateClusterName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("cluster name is required")
	}
	if len(name) < 3 {
		return fmt.Errorf("cluster name must be at least 3 characters")
	}
	if len(name) > 54 {
		return fmt.Errorf("cluster name must be 54 characters or less")
	}
	if !strings.HasPrefix(name, strings.ToLower(string(name[0]))) {
		return fmt.Errorf("cluster name must start with a lowercase letter")
	}
	// Add more validation rules as needed
	return nil
}

// ValidateBillingAccount validates a billing account ID
func ValidateBillingAccount(account string) error {
	if account == "" {
		return nil // Optional field
	}
	if len(account) != 12 {
		return fmt.Errorf("AWS account ID must be 12 digits")
	}
	// Add more validation as needed
	return nil
}
