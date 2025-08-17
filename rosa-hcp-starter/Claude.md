# Claude Development Guide for ROSA HCP CLI

## 🎯 Quick Context

You're working on a **complete rewrite** of the ROSA CLI that supports **ONLY HCP (Hosted Control Plane) clusters**. This is located in `rosa-hcp-starter/` on branch `v2-hcp-only`. The CLI is **99% feature-complete** and **compiles successfully**.

### Key Facts
- **Command Name**: Still `rosa` (not `rosa-hcp`) for compatibility
- **Architecture**: HCP-only, no Classic ROSA support
- **SDK Version**: Uses same OCM SDK as original (v0.1.465)
- **Status**: ✅ Builds successfully, ready for testing

## 📁 Project Structure

```
rosa-hcp-starter/
├── cmd/rosa/           # CLI commands (Cobra)
├── pkg/                # Service layers
├── internal/           # Internal packages
├── go.mod             # Module: github.com/openshift/rosa-hcp
└── rosa               # Compiled binary
```

### Key Services (`pkg/`)
- `cluster/` - Cluster CRUD operations
- `nodepool/` - NodePool management
- `externalauthprovider/` - External OIDC auth
- `breakglass/` - Emergency access credentials
- `iam/` - AWS IAM role management
- `network/` - VPC operations
- `admin/` - Admin user management (HTPasswd)
- `idp/` - Identity providers
- `kubeletconfig/` - Kubelet configuration
- `tuningconfig/` - Performance tuning (HCP-only feature!)
- `ingress/` - Ingress controllers
- `addon/` - Managed services

## 🏗️ Architecture Decisions

### 1. **Modern Go Patterns**
```go
// Result types for error handling
type Result[T any] struct {
    Value T
    Error error
}

// Lazy service initialization
services := initializeServices(ctx, cfg, logger)
```

### 2. **Charm Libraries for UX**
- `huh` - Interactive forms
- `lipgloss` - Styled output
- `bubbletea` - Progress indicators

### 3. **Structured Logging**
Using `log/slog` throughout for structured logs.

### 4. **Service Layer Pattern**
Each package has:
- `Service` interface
- `service` struct (private)
- `NewService()` constructor
- Domain types separate from API types

## ⚠️ Critical Knowledge

### OCM SDK Gotchas

1. **External Auth - Use Correct Types**
```go
// ✅ CORRECT
cmv1.NewExternalAuth()  // Individual provider
.ExternalAuthConfig().ExternalAuths().Add()  // API path

// ❌ WRONG
cmv1.NewExternalAuthConfig()  // Cluster-level config
```

2. **Break-glass - No Individual Delete**
```go
// ✅ API only supports bulk deletion
.BreakGlassCredentials().Delete()  // Deletes ALL

// ❌ NOT SUPPORTED
.BreakGlassCredential(id).Delete()  // No individual
```

3. **Method Names Are Precise**
```go
// ✅ CORRECT
mappings.UserName()  // Capital N
cmv1.NewUsernameClaim()

// ❌ WRONG  
mappings.Username()
cmv1.NewUsernameClaimMapping()
```

4. **Time Comparisons**
```go
// ✅ CORRECT
if !timestamp.IsZero()

// ❌ WRONG
if timestamp != nil  // time.Time is not a pointer
```

### HCP-Specific Features

1. **TuningConfigs** - Performance tuning via TuneD (HCP-ONLY!)
2. **All clusters are HCP** - No Classic support
3. **Requires existing VPC** - No VPC creation in cluster create
4. **External auth for break-glass** - Required for emergency access

### Known Limitations

1. **Shared VPC/Private Hosted Zone** - In private preview, using tags as workaround:
```go
// Temporary workaround
Tags: map[string]string{
    "rosa:shared-vpc-role-arn": roleARN,
}
```

2. **Break-glass deletion** - Only bulk delete, not individual

3. **Some claim mappings** - Simplified implementation

## 🧪 Testing the CLI

### Build
```bash
cd rosa-hcp-starter
go build ./cmd/rosa
./rosa --help
```

### Common Test Commands
```bash
# Authentication
./rosa login --use-auth-code

# Create OIDC config first
./rosa create oidc-config --managed --region us-west-2

# Create cluster (requires existing VPC)
./rosa create cluster --name my-hcp \
  --region us-west-2 \
  --subnet-ids subnet-xxx,subnet-yyy

# NodePool operations
./rosa create nodepool --cluster my-hcp --name workers --replicas 3

# List everything
./rosa list clusters
./rosa list nodepools --cluster my-hcp
```

## 📝 Important Files

### Documentation
- `README.md` - Main documentation (HCP-only focus)
- `SESSION_SUMMARY.md` - Detailed development history
- `OCM_SDK_FIXES.md` - SDK compatibility solutions
- `MISSING_FEATURES.md` - Feature completeness tracking (99%!)
- `HCP_FEATURE_PARITY_ANALYSIS.md` - Comparison with original

### Key Commands to Review
- `cmd/rosa/commands/cluster/create.go` - Most complex command
- `pkg/cluster/service.go` - Core cluster operations
- `pkg/externalauthprovider/service.go` - Fixed SDK usage example
- `pkg/breakglass/service.go` - API limitation handling

## 🔧 Common Development Tasks

### Adding a New Command
1. Create service in `pkg/yourservice/`
2. Create command in `cmd/rosa/commands/yourcommand/`
3. Register in `cmd/rosa/commands/root.go`
4. Follow existing patterns (see `nodepool/` for good example)

### Handling OCM SDK Issues
1. Check original ROSA CLI first - it usually has the answer
2. Look for similar patterns in existing services
3. Remember: method names are precise (`UserName` vs `Username`)
4. Time fields use `.IsZero()` not nil checks

### Output Formatting
```go
writer := output.NewWriter(output.FormatText)
writer.Title("Your Title")
writer.KeyValue(map[string]string{"Key": "Value"})
writer.Success("Operation completed")
writer.Error("Something failed: %v", err)
```

## 🚀 Current State

### ✅ What Works
- All 60+ commands implemented
- Builds successfully
- Core functionality complete
- Modern Go patterns throughout
- Interactive mode with Charm libraries

### 🔄 Needs Testing
- Real cluster operations
- External auth provider flow
- Break-glass credential lifecycle
- Add-on installations
- Upgrade operations

### 📋 Potential Improvements
1. Add unit tests
2. Add integration tests
3. Improve error messages
4. Add command aliases for compatibility
5. Add shell completion
6. Performance optimizations

## 💡 Tips for Future Development

1. **Always check if HCP cluster first** - Many features are HCP-only
2. **Use dry-run for testing** - `--dry-run` flag available
3. **Check SDK response objects** - Use `.Body()` on responses
4. **Region/Profile from config** - Use `cfg.DefaultRegion` and `cfg.ActiveProfile`
5. **Lazy initialization** - Services only created when needed

## 🐛 Debugging

### Common Issues
1. **"aws field is mandatory"** - Must provide IAM roles for cluster creation
2. **"external auth not enabled"** - Create cluster with `--external-auth-providers-enabled`
3. **Compilation errors** - Usually SDK method name issues (check original ROSA)
4. **Service initialization** - Check constructor parameters match

### Useful Debug Commands
```bash
# Check compilation
go build ./...

# Run with debug logging
ROSA_LOG_LEVEL=debug ./rosa create cluster ...

# Check generated API calls
./rosa create cluster --dry-run ...
```

## 📚 References

- Original ROSA CLI: Parent directory (`../`)
- OCM SDK: `github.com/openshift-online/ocm-sdk-go`
- AWS SDK v2: Used for AWS operations
- Cobra docs: For CLI framework
- Charm docs: For TUI components

## 🎯 Mission Statement

This CLI is a **complete replacement** for the original ROSA CLI, but **ONLY for HCP clusters**. It provides a modern, user-friendly experience with better error messages, interactive modes, and a cleaner codebase. The goal is 100% feature parity for HCP operations with improved UX.

---

**Remember**: When in doubt, check how the original ROSA CLI does it - the patterns are usually correct, we just need to adapt them to our cleaner architecture.
