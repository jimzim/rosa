# Modern Go Architecture for ROSA HCP CLI

## Overview
This document outlines the modern Go patterns and architecture for the HCP-only ROSA CLI rewrite, leveraging Go 1.23 features and contemporary libraries.

## Core Design Principles

### 1. Error Handling
Using wrapped errors and custom error types for better debugging and user experience.

### 2. Context-First
All operations should be context-aware for proper cancellation and timeout handling.

### 3. Functional Options Pattern
For flexible and extensible APIs.

### 4. Dependency Injection
For better testability and modularity.

### 5. Result Types
For operations that can fail, providing better error handling patterns.

## Modern Library Stack

### CLI Framework
- **Base**: `spf13/cobra` (keep for compatibility)
- **Interactive UI**: `charmbracelet/huh` (replace survey)
- **Terminal UI**: `charmbracelet/bubbletea` for complex interactions
- **Styling**: `charmbracelet/lipgloss`
- **Tables**: `charmbracelet/lipgloss` with custom table component
- **Spinners**: `charmbracelet/bubbles/spinner`

### Core Libraries
- **Logging**: `log/slog` (structured logging)
- **HTTP**: Standard library with `context`
- **Configuration**: `spf13/viper` with YAML support
- **Validation**: `go-playground/validator/v10`
- **Testing**: `stretchr/testify` + `gomock`

## Project Structure

```
rosa-hcp/
├── cmd/
│   └── rosa/
│       ├── main.go                    # Entry point
│       └── commands/
│           ├── root.go                 # Root command setup
│           ├── cluster/
│           │   ├── create.go
│           │   ├── delete.go
│           │   ├── describe.go
│           │   └── list.go
│           ├── nodepool/
│           │   ├── create.go
│           │   ├── delete.go
│           │   ├── edit.go
│           │   └── list.go
│           └── auth/
│               ├── login.go
│               └── setup.go
├── pkg/
│   ├── api/
│   │   ├── client.go                  # OCM client wrapper
│   │   ├── types.go                   # API types
│   │   └── errors.go                  # API error handling
│   ├── aws/
│   │   ├── client.go                  # AWS client wrapper
│   │   ├── sts.go                     # STS operations
│   │   └── iam.go                     # IAM operations
│   ├── cluster/
│   │   ├── service.go                 # Cluster business logic
│   │   ├── validator.go               # Cluster validation
│   │   └── builder.go                 # Cluster builder pattern
│   ├── nodepool/
│   │   ├── service.go                 # NodePool business logic
│   │   └── validator.go               # NodePool validation
│   ├── interactive/
│   │   ├── prompts.go                 # Interactive prompts
│   │   └── forms.go                   # Complex forms
│   ├── output/
│   │   ├── writer.go                  # Output formatting
│   │   ├── table.go                   # Table rendering
│   │   └── progress.go                # Progress indicators
│   └── errors/
│       ├── errors.go                  # Custom error types
│       └── handler.go                 # Error handling utilities
├── internal/
│   ├── config/
│   │   ├── config.go                  # Configuration management
│   │   └── loader.go                  # Config file loading
│   ├── telemetry/
│   │   └── metrics.go                 # Usage metrics
│   └── version/
│       └── version.go                 # Version information
└── tests/
    ├── integration/
    ├── e2e/
    └── testutil/
        └── mocks/
```

## Code Examples

### 1. Modern Error Handling

```go
// pkg/errors/errors.go
package errors

import (
    "errors"
    "fmt"
)

// Error kinds for categorization
type Kind int

const (
    KindValidation Kind = iota
    KindAPI
    KindAWS
    KindPermission
    KindNetwork
    KindNotFound
)

// ROSAError provides structured errors with context
type ROSAError struct {
    Op         string                 // Operation being performed
    Kind       Kind                   // Error category
    Err        error                  // Underlying error
    Suggestion string                 // User-actionable suggestion
    Details    map[string]interface{} // Additional context
}

func (e *ROSAError) Error() string {
    if e.Suggestion != "" {
        return fmt.Sprintf("%s: %v\n💡 Suggestion: %s", e.Op, e.Err, e.Suggestion)
    }
    return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *ROSAError) Unwrap() error {
    return e.Err
}

// Helper constructors
func Validation(op string, err error) *ROSAError {
    return &ROSAError{
        Op:   op,
        Kind: KindValidation,
        Err:  err,
    }
}

func WithSuggestion(err *ROSAError, suggestion string) *ROSAError {
    err.Suggestion = suggestion
    return err
}

// Result type for operations that can fail
type Result[T any] struct {
    value T
    err   error
}

func Ok[T any](value T) Result[T] {
    return Result[T]{value: value}
}

func Err[T any](err error) Result[T] {
    return Result[T]{err: err}
}

func (r Result[T]) Unwrap() (T, error) {
    return r.value, r.err
}

func (r Result[T]) IsOk() bool {
    return r.err == nil
}
```

### 2. Modern CLI Structure

```go
// cmd/rosa/commands/cluster/create.go
package cluster

import (
    "context"
    "fmt"
    
    "github.com/spf13/cobra"
    "github.com/charmbracelet/huh"
    "github.com/openshift/rosa-hcp/pkg/cluster"
    "github.com/openshift/rosa-hcp/pkg/interactive"
    "github.com/openshift/rosa-hcp/pkg/output"
)

type CreateOptions struct {
    Name           string
    Region         string
    Version        string
    NodePools      []NodePoolConfig
    MultiAZ        bool
    PrivateLink    bool
    BillingAccount string
    Tags           map[string]string
    
    // Flags
    Interactive bool
    Output      string
    DryRun      bool
}

func NewCreateCommand(srv *cluster.Service) *cobra.Command {
    opts := &CreateOptions{}
    
    cmd := &cobra.Command{
        Use:   "create",
        Short: "Create a new ROSA HCP cluster",
        Long:  "Create a new Red Hat OpenShift Service on AWS (ROSA) cluster with Hosted Control Planes",
        Example: `  # Create a cluster interactively
  rosa create cluster --interactive
  
  # Create a cluster with specific configuration
  rosa create cluster --name my-cluster --region us-west-2 --version 4.14.0`,
        RunE: func(cmd *cobra.Command, args []string) error {
            return runCreate(cmd.Context(), srv, opts)
        },
    }
    
    // Flags using functional options pattern
    flags := cmd.Flags()
    flags.StringVar(&opts.Name, "name", "", "Name of the cluster")
    flags.StringVar(&opts.Region, "region", "", "AWS region for the cluster")
    flags.StringVar(&opts.Version, "version", "", "OpenShift version")
    flags.BoolVar(&opts.MultiAZ, "multi-az", true, "Deploy to multiple availability zones")
    flags.BoolVar(&opts.PrivateLink, "private-link", false, "Create cluster with AWS PrivateLink")
    flags.StringVar(&opts.BillingAccount, "billing-account", "", "AWS billing account ID")
    flags.StringToStringVar(&opts.Tags, "tags", nil, "Tags to apply to the cluster")
    
    flags.BoolVarP(&opts.Interactive, "interactive", "i", false, "Interactive mode")
    flags.StringVarP(&opts.Output, "output", "o", "text", "Output format (text|json|yaml)")
    flags.BoolVar(&opts.DryRun, "dry-run", false, "Simulate cluster creation")
    
    return cmd
}

func runCreate(ctx context.Context, srv *cluster.Service, opts *CreateOptions) error {
    // Modern interactive mode using Charm's huh
    if opts.Interactive {
        if err := interactiveCreate(ctx, opts); err != nil {
            return fmt.Errorf("interactive create failed: %w", err)
        }
    }
    
    // Validation with structured errors
    if err := validateCreateOptions(opts); err != nil {
        return err
    }
    
    // Progress indicator
    progress := output.NewProgress("Creating cluster")
    progress.Start()
    defer progress.Stop()
    
    // Create cluster using service layer
    config := cluster.CreateConfig{
        Name:           opts.Name,
        Region:         opts.Region,
        Version:        opts.Version,
        MultiAZ:        opts.MultiAZ,
        PrivateLink:    opts.PrivateLink,
        BillingAccount: opts.BillingAccount,
        Tags:           opts.Tags,
    }
    
    result := srv.Create(ctx, config)
    cluster, err := result.Unwrap()
    if err != nil {
        return handleCreateError(err)
    }
    
    progress.Success("Cluster created successfully")
    
    // Output formatting
    return output.Print(cluster, opts.Output)
}

func interactiveCreate(ctx context.Context, opts *CreateOptions) error {
    form := huh.NewForm(
        huh.NewGroup(
            huh.NewInput().
                Title("Cluster Name").
                Value(&opts.Name).
                Validate(validateClusterName),
            
            huh.NewSelect[string]().
                Title("AWS Region").
                Options(
                    huh.NewOption("US East (N. Virginia)", "us-east-1"),
                    huh.NewOption("US West (Oregon)", "us-west-2"),
                    huh.NewOption("EU (Ireland)", "eu-west-1"),
                ).
                Value(&opts.Region),
            
            huh.NewConfirm().
                Title("Enable PrivateLink?").
                Description("Use AWS PrivateLink for private connectivity").
                Value(&opts.PrivateLink),
        ),
    ).WithTheme(huh.ThemeCharm())
    
    return form.Run()
}
```

### 3. Service Layer with Dependency Injection

```go
// pkg/cluster/service.go
package cluster

import (
    "context"
    "fmt"
    
    "github.com/openshift/rosa-hcp/pkg/api"
    "github.com/openshift/rosa-hcp/pkg/aws"
    "github.com/openshift/rosa-hcp/pkg/errors"
    "log/slog"
)

// Service provides cluster operations
type Service struct {
    ocm    api.Client
    aws    aws.Client
    logger *slog.Logger
}

// NewService creates a new cluster service
func NewService(ocm api.Client, aws aws.Client, logger *slog.Logger) *Service {
    return &Service{
        ocm:    ocm,
        aws:    aws,
        logger: logger.With("service", "cluster"),
    }
}

// CreateConfig holds cluster creation configuration
type CreateConfig struct {
    Name           string
    Region         string
    Version        string
    MultiAZ        bool
    PrivateLink    bool
    BillingAccount string
    Tags           map[string]string
}

// Create creates a new HCP cluster
func (s *Service) Create(ctx context.Context, config CreateConfig) errors.Result[*Cluster] {
    s.logger.InfoContext(ctx, "creating cluster",
        slog.String("name", config.Name),
        slog.String("region", config.Region),
    )
    
    // Build cluster using builder pattern
    builder := NewClusterBuilder().
        WithName(config.Name).
        WithRegion(config.Region).
        WithVersion(config.Version).
        WithMultiAZ(config.MultiAZ).
        WithPrivateLink(config.PrivateLink).
        WithTags(config.Tags)
    
    if config.BillingAccount != "" {
        builder.WithBillingAccount(config.BillingAccount)
    }
    
    clusterSpec, err := builder.Build()
    if err != nil {
        return errors.Err[*Cluster](
            errors.Validation("cluster.Create", err).
                WithSuggestion("Check cluster configuration requirements"),
        )
    }
    
    // Create via OCM API
    created, err := s.ocm.CreateCluster(ctx, clusterSpec)
    if err != nil {
        s.logger.ErrorContext(ctx, "failed to create cluster",
            slog.String("error", err.Error()),
        )
        return errors.Err[*Cluster](
            errors.API("cluster.Create", err),
        )
    }
    
    s.logger.InfoContext(ctx, "cluster created successfully",
        slog.String("id", created.ID),
    )
    
    return errors.Ok(created)
}
```

### 4. Modern Configuration Management

```go
// internal/config/config.go
package config

import (
    "fmt"
    "os"
    "path/filepath"
    
    "github.com/spf13/viper"
)

type Config struct {
    DefaultRegion string            `mapstructure:"default_region"`
    Profiles      map[string]Profile `mapstructure:"profiles"`
    ActiveProfile string            `mapstructure:"active_profile"`
}

type Profile struct {
    Name      string    `mapstructure:"name"`
    Region    string    `mapstructure:"region"`
    AccountID string    `mapstructure:"account_id"`
    STS       STSConfig `mapstructure:"sts"`
}

type STSConfig struct {
    RoleARN        string `mapstructure:"role_arn"`
    SupportRoleARN string `mapstructure:"support_role_arn"`
    WorkerRoleARN  string `mapstructure:"worker_role_arn"`
}

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
    viper.SetDefault("default_region", "us-west-2")
    
    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return nil, fmt.Errorf("failed to read config: %w", err)
        }
    }
    
    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %w", err)
    }
    
    return &config, nil
}
```

### 5. Modern Output Formatting

```go
// pkg/output/writer.go
package output

import (
    "encoding/json"
    "fmt"
    "io"
    "os"
    
    "github.com/charmbracelet/lipgloss"
    "gopkg.in/yaml.v3"
)

type Format string

const (
    FormatText Format = "text"
    FormatJSON Format = "json"
    FormatYAML Format = "yaml"
)

type Writer struct {
    out    io.Writer
    format Format
    style  lipgloss.Style
}

func NewWriter(format Format) *Writer {
    return &Writer{
        out:    os.Stdout,
        format: format,
        style:  lipgloss.NewStyle(),
    }
}

func (w *Writer) Write(v interface{}) error {
    switch w.format {
    case FormatJSON:
        return w.writeJSON(v)
    case FormatYAML:
        return w.writeYAML(v)
    default:
        return w.writeText(v)
    }
}

func (w *Writer) writeJSON(v interface{}) error {
    encoder := json.NewEncoder(w.out)
    encoder.SetIndent("", "  ")
    return encoder.Encode(v)
}

func (w *Writer) writeYAML(v interface{}) error {
    encoder := yaml.NewEncoder(w.out)
    defer encoder.Close()
    return encoder.Encode(v)
}

func (w *Writer) writeText(v interface{}) error {
    // Use lipgloss for beautiful text output
    switch val := v.(type) {
    case fmt.Stringer:
        fmt.Fprintln(w.out, val.String())
    default:
        fmt.Fprintf(w.out, "%+v\n", val)
    }
    return nil
}

// Table creates a styled table
func (w *Writer) Table(headers []string, rows [][]string) {
    table := lipgloss.NewStyle().
        BorderStyle(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("99"))
    
    // Render table using lipgloss
    // ... table rendering logic ...
}
```

## Testing Strategy

### Unit Testing with Modern Patterns

```go
// pkg/cluster/service_test.go
package cluster_test

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "go.uber.org/mock/gomock"
    
    "github.com/openshift/rosa-hcp/pkg/cluster"
    "github.com/openshift/rosa-hcp/pkg/api/mocks"
)

func TestService_Create(t *testing.T) {
    tests := []struct {
        name    string
        config  cluster.CreateConfig
        setup   func(*mocks.MockClient)
        wantErr bool
    }{
        {
            name: "successful creation",
            config: cluster.CreateConfig{
                Name:    "test-cluster",
                Region:  "us-west-2",
                Version: "4.14.0",
            },
            setup: func(m *mocks.MockClient) {
                m.EXPECT().
                    CreateCluster(gomock.Any(), gomock.Any()).
                    Return(&api.Cluster{ID: "123"}, nil)
            },
            wantErr: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()
            
            mockClient := mocks.NewMockClient(ctrl)
            tt.setup(mockClient)
            
            svc := cluster.NewService(mockClient, nil, slog.Default())
            result := svc.Create(context.Background(), tt.config)
            
            if tt.wantErr {
                assert.False(t, result.IsOk())
            } else {
                assert.True(t, result.IsOk())
                cluster, err := result.Unwrap()
                require.NoError(t, err)
                assert.NotNil(t, cluster)
            }
        })
    }
}
```

## Build and Distribution

### Makefile for Modern Build

```makefile
.PHONY: build
build:
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o bin/rosa ./cmd/rosa

.PHONY: test
test:
	go test -race -coverprofile=coverage.out ./...

.PHONY: lint
lint:
	golangci-lint run --fix

.PHONY: install-tools
install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install go.uber.org/mock/mockgen@latest

.PHONY: generate
generate:
	go generate ./...
```

## Performance Optimizations

1. **Connection Pooling**: Reuse HTTP connections
2. **Concurrent Operations**: Use goroutines for parallel API calls
3. **Lazy Loading**: Load configuration only when needed
4. **Binary Size**: Use `-ldflags="-s -w"` to strip debug info

## Conclusion

This modern Go architecture provides:
- Better error handling with structured errors
- Beautiful CLI experience with Charm libraries
- Testable code with dependency injection
- Type-safe configuration management
- Context-aware operations throughout
- Clean separation of concerns
