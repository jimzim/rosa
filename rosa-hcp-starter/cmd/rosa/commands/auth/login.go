package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/pkg/api"
	"github.com/openshift/rosa-hcp/pkg/interactive"
	"github.com/openshift/rosa-hcp/pkg/output"
)

// LoginOptions contains the options for the login command
type LoginOptions struct {
	Token        string
	ClientID     string
	ClientSecret string
	UseAuthCode  bool
	Insecure     bool
	OCMEnv       string
	RedirectURL  string
}

// OAuth endpoints for different environments
var oauthEndpoints = map[string]string{
	"production":  "https://sso.redhat.com/auth/realms/redhat-external/protocol/openid-connect",
	"staging":     "https://sso.stage.redhat.com/auth/realms/redhat-external/protocol/openid-connect",
	"integration": "https://sso.integration.redhat.com/auth/realms/redhat-external/protocol/openid-connect",
}

// Default OAuth client ID for ROSA
const defaultClientID = "ocm-cli"

// NewLoginCommand creates the login command
func NewLoginCommand() *cobra.Command {
	opts := &LoginOptions{}

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to your Red Hat account",
		Long: `Log in to your Red Hat account to manage ROSA clusters.

You can log in using:
1. Browser-based authentication (recommended)
2. An offline access token
3. Environment variables (ROSA_TOKEN)`,
		Example: `  # Login with browser (recommended)
  rosa login --use-auth-code
  
  # Login with token
  rosa login --token $OFFLINE_ACCESS_TOKEN
  
  # Login interactively
  rosa login`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(cmd.Context(), opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.Token, "token", "", "Offline access token")
	flags.BoolVar(&opts.UseAuthCode, "use-auth-code", false, "Use browser-based authentication (recommended)")
	flags.StringVar(&opts.ClientID, "client-id", defaultClientID, "OAuth client ID")
	flags.StringVar(&opts.ClientSecret, "client-secret", "", "OAuth client secret")
	flags.BoolVar(&opts.Insecure, "insecure", false, "Skip TLS certificate verification")
	flags.StringVar(&opts.OCMEnv, "env", "production", "Environment (production, staging, integration)")

	return cmd
}

func runLogin(ctx context.Context, opts *LoginOptions) error {
	// If use-auth-code is specified, use browser-based flow
	if opts.UseAuthCode {
		return runAuthCodeLogin(ctx, opts)
	}

	// Check for token in environment if not provided
	if opts.Token == "" && opts.ClientID == "" {
		opts.Token = os.Getenv("ROSA_TOKEN")
		if opts.Token == "" {
			opts.Token = os.Getenv("OCM_TOKEN")
		}
	}

	// Interactive mode if no credentials provided
	if opts.Token == "" && opts.ClientID == defaultClientID && opts.ClientSecret == "" {
		if err := runInteractiveLogin(opts); err != nil {
			return err
		}
	}

	// If still no auth method selected, default to auth code
	if opts.Token == "" && opts.ClientSecret == "" {
		output.Info("No authentication method specified. Using browser-based authentication.")
		return runAuthCodeLogin(ctx, opts)
	}

	// Validate credentials
	if opts.Token == "" && (opts.ClientID == "" || opts.ClientSecret == "") {
		return fmt.Errorf("either a token or client credentials are required")
	}

	// Determine API URL based on environment
	apiURL := getAPIURL(opts.OCMEnv)

	// Test the connection
	progress := output.NewProgress("Logging in to Red Hat account")
	progress.Start()
	defer progress.Stop()

	apiConfig := api.Config{
		URL:          apiURL,
		Token:        opts.Token,
		ClientID:     opts.ClientID,
		ClientSecret: opts.ClientSecret,
	}

	// Try to create a connection
	client, err := api.NewClient(ctx, apiConfig)
	if err != nil {
		progress.Stop()
		return fmt.Errorf("failed to log in: %w", err)
	}

	// Get current account info to verify login
	conn := client.GetConnection()
	if conn == nil {
		progress.Stop()
		return fmt.Errorf("failed to establish connection")
	}

	// Get account info
	accountResp, err := conn.AccountsMgmt().V1().CurrentAccount().Get().Send()
	if err != nil {
		progress.Stop()
		return fmt.Errorf("failed to get account information: %w", err)
	}

	account := accountResp.Body()
	progress.Success("Logged in successfully")

	// Save configuration
	cfg := &config.Config{
		APIURL:        apiURL,
		Token:         opts.Token,
		DefaultRegion: "us-west-2",
	}

	if err := cfg.Save(); err != nil {
		output.Warning("Failed to save configuration: %v", err)
	}

	// Display account info
	displayAccountInfo(account, opts.OCMEnv)

	return nil
}

// runAuthCodeLogin performs browser-based authentication using OAuth authorization code flow
func runAuthCodeLogin(ctx context.Context, opts *LoginOptions) error {
	// Generate state for CSRF protection
	state := generateState()

	// Generate PKCE challenge
	codeVerifier := generateCodeVerifier()
	codeChallenge := generateCodeChallenge(codeVerifier)

	// Start local HTTP server to receive the callback
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to start callback server: %w", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	redirectURL := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	// Build authorization URL
	authURL := buildAuthURL(opts.OCMEnv, opts.ClientID, redirectURL, state, codeChallenge)

	// Channel to receive the authorization code
	codeChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	// Start HTTP server in background
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/callback" {
				http.NotFound(w, r)
				return
			}

			// Check state
			if r.URL.Query().Get("state") != state {
				errorChan <- fmt.Errorf("invalid state parameter")
				http.Error(w, "Invalid state", http.StatusBadRequest)
				return
			}

			// Check for error
			if errParam := r.URL.Query().Get("error"); errParam != "" {
				errorChan <- fmt.Errorf("authentication failed: %s", errParam)
				http.Error(w, "Authentication failed", http.StatusBadRequest)
				return
			}

			// Get authorization code
			code := r.URL.Query().Get("code")
			if code == "" {
				errorChan <- fmt.Errorf("no authorization code received")
				http.Error(w, "No authorization code", http.StatusBadRequest)
				return
			}

			// Send success page
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <title>ROSA Login Successful</title>
    <style>
        body { font-family: sans-serif; text-align: center; padding: 50px; }
        .success { color: #28a745; font-size: 24px; }
        .message { margin-top: 20px; color: #666; }
    </style>
</head>
<body>
    <div class="success">✅ Login Successful!</div>
    <div class="message">You can now close this window and return to your terminal.</div>
</body>
</html>`)

			codeChan <- code
		}),
	}

	// Start server in goroutine
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errorChan <- err
		}
	}()

	// Open browser
	output.Info("Opening browser for authentication...")
	output.Info("If the browser doesn't open automatically, visit:")
	output.Info(authURL)
	output.Info("")

	if err := openBrowser(authURL); err != nil {
		output.Warning("Failed to open browser automatically: %v", err)
	}

	// Wait for callback
	output.Info("Waiting for authentication...")

	var authCode string
	select {
	case authCode = <-codeChan:
		// Success
	case err := <-errorChan:
		server.Close()
		return fmt.Errorf("authentication failed: %w", err)
	case <-time.After(5 * time.Minute):
		server.Close()
		return fmt.Errorf("authentication timed out")
	case <-ctx.Done():
		server.Close()
		return ctx.Err()
	}

	// Shutdown server
	server.Close()

	// Exchange authorization code for tokens
	output.Info("Exchanging authorization code for tokens...")
	tokens, err := exchangeAuthCode(ctx, opts.OCMEnv, opts.ClientID, authCode, codeVerifier, redirectURL)
	if err != nil {
		return fmt.Errorf("failed to exchange authorization code: %w", err)
	}

	// Use the access token to authenticate with OCM
	apiURL := getAPIURL(opts.OCMEnv)

	apiConfig := api.Config{
		URL:          apiURL,
		Token:        tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}

	// Create OCM connection
	client, err := api.NewClient(ctx, apiConfig)
	if err != nil {
		return fmt.Errorf("failed to create OCM client: %w", err)
	}

	// Verify login
	conn := client.GetConnection()
	if conn == nil {
		return fmt.Errorf("failed to establish connection")
	}

	accountResp, err := conn.AccountsMgmt().V1().CurrentAccount().Get().Send()
	if err != nil {
		return fmt.Errorf("failed to get account information: %w", err)
	}

	account := accountResp.Body()

	// Save configuration with tokens
	cfg := &config.Config{
		APIURL:        apiURL,
		Token:         tokens.AccessToken,
		RefreshToken:  tokens.RefreshToken,
		DefaultRegion: "us-west-2",
	}

	if err := cfg.Save(); err != nil {
		output.Warning("Failed to save configuration: %v", err)
	}

	output.Success("Logged in successfully!")
	displayAccountInfo(account, opts.OCMEnv)

	return nil
}

// TokenResponse represents the OAuth token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// exchangeAuthCode exchanges the authorization code for tokens
func exchangeAuthCode(ctx context.Context, env, clientID, code, codeVerifier, redirectURL string) (*TokenResponse, error) {
	endpoint := oauthEndpoints[env]
	tokenURL := fmt.Sprintf("%s/token", endpoint)

	// Build form data
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", clientID)
	data.Set("code", code)
	data.Set("code_verifier", codeVerifier)
	data.Set("redirect_uri", redirectURL)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the JSON response
	var tokens TokenResponse
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &tokens, nil
}

// buildAuthURL builds the OAuth authorization URL
func buildAuthURL(env, clientID, redirectURL, state, codeChallenge string) string {
	endpoint := oauthEndpoints[env]
	authURL := fmt.Sprintf("%s/auth", endpoint)

	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("redirect_uri", redirectURL)
	params.Set("response_type", "code")
	params.Set("scope", "openid")
	params.Set("state", state)
	params.Set("code_challenge", codeChallenge)
	params.Set("code_challenge_method", "S256")

	return fmt.Sprintf("%s?%s", authURL, params.Encode())
}

// generateState generates a random state for CSRF protection
func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	// Use RawURLEncoding to avoid padding issues
	return base64.RawURLEncoding.EncodeToString(b)
}

// generateCodeVerifier generates a PKCE code verifier
func generateCodeVerifier() string {
	// Generate 32 random bytes (43-128 characters when base64url encoded)
	b := make([]byte, 32)
	rand.Read(b)
	// Use RawURLEncoding to avoid padding
	return base64.RawURLEncoding.EncodeToString(b)
}

// generateCodeChallenge generates a PKCE code challenge from the verifier
func generateCodeChallenge(verifier string) string {
	// Create SHA256 hash of the verifier
	h := sha256.New()
	h.Write([]byte(verifier))
	// Use RawURLEncoding (no padding) for the challenge
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// openBrowser opens the URL in the default browser
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default:
		return fmt.Errorf("unsupported platform")
	}

	return exec.Command(cmd, args...).Start()
}

// displayAccountInfo displays the account information
func displayAccountInfo(account interface{}, env string) {
	// This is a simplified version - in production you'd properly handle the account type
	output.Info("\nLogged in as:")
	output.Info("  Environment: %s", env)
	output.Info("\nYou can now create and manage ROSA HCP clusters!")
}

func runInteractiveLogin(opts *LoginOptions) error {
	output.Info("Log in to Red Hat OpenShift Cluster Manager")
	output.Info("")

	// Choose login method
	methods := []interactive.SelectOption[string]{
		{Label: "Browser Authentication (recommended)", Value: "browser"},
		{Label: "Token", Value: "token"},
		{Label: "Exit", Value: "exit"},
	}

	method, err := interactive.PromptSelect(
		"How would you like to log in?",
		"Choose your authentication method",
		methods,
		"browser",
	)
	if err != nil {
		return err
	}

	switch method {
	case "browser":
		opts.UseAuthCode = true

	case "token":
		output.Info("\nTo get an offline access token, visit:")
		output.Info("https://console.redhat.com/openshift/token/rosa")
		output.Info("")

		token, err := interactive.PromptString(
			"Offline Access Token",
			"Paste your token",
			"",
			true,
		)
		if err != nil {
			return err
		}
		opts.Token = strings.TrimSpace(token)

	case "exit":
		return fmt.Errorf("login cancelled")
	}

	// Select environment
	envs := []interactive.SelectOption[string]{
		{Label: "Production (default)", Value: "production"},
		{Label: "Staging", Value: "staging"},
		{Label: "Integration", Value: "integration"},
	}

	env, err := interactive.PromptSelect(
		"Environment",
		"Select the OCM environment",
		envs,
		"production",
	)
	if err != nil {
		return err
	}
	opts.OCMEnv = env

	return nil
}

func getAPIURL(env string) string {
	switch env {
	case "staging":
		return "https://api.stage.openshift.com"
	case "integration":
		return "https://api.integration.openshift.com"
	default:
		return "https://api.openshift.com"
	}
}
