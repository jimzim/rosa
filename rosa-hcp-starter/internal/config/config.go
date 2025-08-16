package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the CLI configuration
type Config struct {
	APIURL        string             `mapstructure:"api_url"`
	Token         string             `mapstructure:"token"`
	RefreshToken  string             `mapstructure:"refresh_token"`
	DefaultRegion string             `mapstructure:"default_region"`
	Profiles      map[string]Profile `mapstructure:"profiles"`
	ActiveProfile string             `mapstructure:"active_profile"`
}

// Profile represents a configuration profile
type Profile struct {
	Name      string    `mapstructure:"name"`
	Region    string    `mapstructure:"region"`
	AccountID string    `mapstructure:"account_id"`
	STS       STSConfig `mapstructure:"sts"`
}

// STSConfig represents STS configuration for HCP
type STSConfig struct {
	RoleARN        string `mapstructure:"role_arn"`
	SupportRoleARN string `mapstructure:"support_role_arn"`
	WorkerRoleARN  string `mapstructure:"worker_role_arn"`
	ExternalID     string `mapstructure:"external_id"`
}

// Default returns a default configuration
func Default() *Config {
	return &Config{
		APIURL:        "https://api.openshift.com",
		DefaultRegion: "us-west-2",
		Profiles:      make(map[string]Profile),
	}
}

// Load loads configuration from file
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Config paths
	if configDir := os.Getenv("ROSA_CONFIG_DIR"); configDir != "" {
		viper.AddConfigPath(configDir)
	}

	home, err := os.UserHomeDir()
	if err == nil {
		viper.AddConfigPath(filepath.Join(home, ".rosa"))
	}

	viper.AddConfigPath(".")

	// Environment variables
	viper.SetEnvPrefix("ROSA")
	viper.AutomaticEnv()

	// Defaults
	viper.SetDefault("api_url", "https://api.openshift.com")
	viper.SetDefault("default_region", "us-west-2")

	// Try to read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; use defaults
			return Default(), nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Load token from environment if not in config
	if config.Token == "" {
		config.Token = os.Getenv("ROSA_TOKEN")
	}

	return &config, nil
}

// GetProfile returns the specified profile or the active profile
func (c *Config) GetProfile(name string) Profile {
	if name != "" {
		if profile, ok := c.Profiles[name]; ok {
			return profile
		}
	}

	if c.ActiveProfile != "" {
		if profile, ok := c.Profiles[c.ActiveProfile]; ok {
			return profile
		}
	}

	// Return default profile
	return Profile{
		Name:   "default",
		Region: c.DefaultRegion,
	}
}

// Save saves the configuration to file
func (c *Config) Save() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".rosa")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configFile := filepath.Join(configDir, "config.yaml")

	viper.Set("api_url", c.APIURL)
	viper.Set("token", c.Token)
	viper.Set("refresh_token", c.RefreshToken)
	viper.Set("default_region", c.DefaultRegion)
	viper.Set("profiles", c.Profiles)
	viper.Set("active_profile", c.ActiveProfile)

	if err := viper.WriteConfigAs(configFile); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}
