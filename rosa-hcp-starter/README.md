# ROSA HCP CLI - MVP Implementation

## Overview

This is the MVP implementation of the HCP-only ROSA CLI v2.0. This version exclusively supports ROSA clusters with Hosted Control Planes (HCP), removing all Classic cluster support for a cleaner, more maintainable codebase.

## Key Differences from v1.x

| Feature | v1.x (Classic + HCP) | v2.x (HCP-Only) |
|---------|---------------------|-----------------|
| **Cluster Type** | Requires `--hosted-cp` flag | All clusters are HCP |
| **IAM Roles** | 4 roles (including Control Plane) | 3 roles only |
| **Node Management** | MachinePools and NodePools | NodePools only |
| **Default Architecture** | Classic | HCP |
| **Codebase Size** | ~100K lines | ~40K lines (estimated) |

## Quick Start

### Prerequisites

1. **Go 1.23+** installed
2. **AWS CLI** configured with credentials
3. **Red Hat account** with ROSA access

### Build

```bash
# Build the CLI
make build

# Or build directly
go build -o bin/rosa ./cmd/rosa
```

### Configuration

Create a configuration file at `~/.rosa/config.yaml`:

```yaml
api_url: https://api.openshift.com
token: your-ocm-token-here
default_region: us-west-2
profiles:
  default:
    region: us-west-2
    sts:
      role_arn: arn:aws:iam::123456789012:role/ManagedOpenShift-HCP-ROSA-Installer-Role
      support_role_arn: arn:aws:iam::123456789012:role/ManagedOpenShift-HCP-ROSA-Support-Role
      worker_role_arn: arn:aws:iam::123456789012:role/ManagedOpenShift-HCP-ROSA-Worker-Role
```

Or use environment variables:

```bash
export ROSA_TOKEN=your-ocm-token
export AWS_REGION=us-west-2
```

### Usage Examples

#### Create a Cluster (Interactive)

```bash
# Interactive mode guides you through all options
./bin/rosa create cluster --interactive
```

#### Create a Cluster (Direct)

```bash
# Minimal configuration
./bin/rosa create cluster \
  --name my-hcp-cluster \
  --region us-west-2

# With specific configuration
./bin/rosa create cluster \
  --name production-cluster \
  --region us-west-2 \
  --version 4.14.0 \
  --compute-nodes 3 \
  --compute-type m5.xlarge \
  --multi-az \
  --private-link
```

#### Create with STS Roles

```bash
./bin/rosa create cluster \
  --name my-cluster \
  --region us-west-2 \
  --role-arn arn:aws:iam::123456789012:role/ManagedOpenShift-HCP-ROSA-Installer-Role \
  --support-role-arn arn:aws:iam::123456789012:role/ManagedOpenShift-HCP-ROSA-Support-Role \
  --worker-iam-role arn:aws:iam::123456789012:role/ManagedOpenShift-HCP-ROSA-Worker-Role
```

## Implementation Status

### ✅ Completed (MVP)

- [x] `rosa create cluster` - HCP-only cluster creation
- [x] Modern CLI framework with Charm libraries
- [x] Interactive mode with beautiful prompts
- [x] OCM API integration
- [x] AWS STS integration (3 roles only)
- [x] Configuration management
- [x] Error handling with suggestions

### 🚧 In Progress

- [ ] `rosa delete cluster`
- [ ] `rosa describe cluster`
- [ ] `rosa list clusters`
- [ ] `rosa create nodepool`
- [ ] `rosa delete nodepool`

### 📋 Planned

- [ ] Upgrade management
- [ ] OIDC provider creation
- [ ] External authentication
- [ ] Tuning configs (HCP-specific)
- [ ] Audit log forwarding
- [ ] Break-glass credentials

## Architecture

```
rosa-hcp-starter/
├── cmd/rosa/
│   ├── main.go                    # Entry point
│   └── commands/
│       ├── root.go                 # Root command
│       ├── cluster/                # Cluster commands
│       │   ├── create.go           # ✅ Implemented
│       │   └── cluster.go
│       └── nodepool/               # NodePool commands
├── pkg/
│   ├── api/                        # OCM API client
│   │   └── client.go               # ✅ Implemented
│   ├── aws/                        # AWS operations
│   │   └── client.go               # ✅ Implemented
│   ├── cluster/                    # Cluster business logic
│   │   └── service.go              # ✅ Implemented
│   ├── nodepool/                   # NodePool logic
│   ├── interactive/                # Interactive prompts
│   │   └── prompts.go              # ✅ Implemented
│   ├── output/                     # Output formatting
│   │   └── writer.go               # ✅ Implemented
│   └── errors/                     # Error handling
│       └── errors.go               # ✅ Implemented
└── internal/
    ├── config/                     # Configuration
    │   └── config.go               # ✅ Implemented
    └── version/                    # Version info
        └── version.go              # ✅ Implemented
```

## Testing

### Unit Tests

```bash
# Run all tests
make test

# Run with coverage
make coverage

# Run specific package tests
go test ./pkg/cluster/...
```

### Integration Tests

```bash
# Test cluster creation (dry-run)
./bin/rosa create cluster \
  --name test-cluster \
  --region us-west-2 \
  --dry-run

# Test with mock OCM API
OCM_API_URL=http://localhost:8080 ./bin/rosa create cluster --name test
```

## Development

### Adding New Commands

1. Create command file in appropriate package:
```go
// cmd/rosa/commands/cluster/describe.go
func NewDescribeCommand(svc *cluster.Service) *cobra.Command {
    // Implementation
}
```

2. Add to parent command:
```go
// cmd/rosa/commands/cluster/cluster.go
cmd.AddCommand(
    NewDescribeCommand(svc),
)
```

3. Implement service logic:
```go
// pkg/cluster/service.go
func (s *Service) Describe(ctx context.Context, clusterID string) (*Cluster, error) {
    // Implementation
}
```

### Code Style

- Use structured logging with `slog`
- Return `errors.Result[T]` for operations that can fail
- Use Charm libraries for all terminal UI
- Context-aware operations throughout
- Comprehensive error messages with suggestions

## Migration from v1.x

### For Users

```bash
# Install v2 (HCP-only)
brew install rosa

# Keep v1 for Classic clusters
brew install rosa@1

# Check version
rosa version
# Output: 2.0.0-dev (HCP-Only Edition)
```

### Breaking Changes

1. **No Classic Support**: Cannot create Classic clusters
2. **No `--hosted-cp` flag**: All clusters are HCP
3. **NodePools only**: No MachinePool commands
4. **3 IAM roles**: Control Plane role not needed
5. **STS only**: No mint mode support

## Contributing

### Development Setup

```bash
# Clone the repo
git clone https://github.com/openshift/rosa.git
cd rosa

# Checkout v2 branch
git checkout v2-hcp-only

# Install dependencies
cd rosa-hcp-starter
go mod download

# Install dev tools
make install-tools
```

### Testing Your Changes

1. Build: `make build`
2. Test: `make test`
3. Lint: `make lint`
4. Format: `make fmt`

## Troubleshooting

### Common Issues

#### Authentication Error
```
Error: failed to create OCM connection
```
**Solution**: Ensure your OCM token is valid:
```bash
rosa login --token $OCM_TOKEN
```

#### AWS Credentials Error
```
Error: failed to validate installer role
```
**Solution**: Check AWS credentials and role permissions:
```bash
aws sts get-caller-identity
rosa verify permissions
```

#### Invalid Cluster Name
```
Error: cluster name must start with a lowercase letter
```
**Solution**: Use lowercase letters, numbers, and hyphens only.

## Support

- **Documentation**: [ROSA HCP Docs](https://docs.openshift.com/rosa/rosa_hcp)
- **Issues**: [GitHub Issues](https://github.com/openshift/rosa/issues)
- **Community**: [OpenShift Slack](https://slack.openshift.io/)

## License

Apache License 2.0
