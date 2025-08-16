# ROSA CLI Migration Strategy: HCP-Only Version

## Recommended Approach: Branch-Based Major Version

### Strategy Overview
- **v1.x branch (main)**: Current CLI supporting both Classic and HCP
- **v2.x branch (v2-hcp-only)**: New HCP-only CLI
- **Parallel maintenance**: v1.x gets security fixes, v2.x gets new features

## Implementation Phases

### Phase 1: Branch Setup (Week 1)
```bash
# Create v2 branch
git checkout -b v2-hcp-only

# Update version to 2.0.0-alpha
```

#### Initial Changes
1. Update README to indicate HCP-only
2. Update go.mod to v2
3. Create MIGRATION.md for users

### Phase 2: Progressive Removal (Weeks 2-4)

#### Step 1: Remove Classic-Specific Commands
```bash
# Commands to remove entirely
rm -rf cmd/create/machinepool/
rm -rf cmd/edit/machinepool/
rm -rf cmd/describe/machinepool/
rm -rf cmd/delete/machinepool/
rm -rf cmd/list/machinepool/
```

#### Step 2: Clean Command Structure
```go
// Before: cmd/create/cluster/cmd.go
if cluster.Hypershift().Enabled() {
    // HCP logic
} else {
    // Classic logic - REMOVE
}

// After: Direct HCP implementation
// HCP logic only
```

#### Step 3: Remove Classic IAM/Roles
- Remove Control Plane role management
- Remove mint mode support
- Simplify to 3 HCP roles only

#### Step 4: Simplify Package Structure
```bash
# Remove Classic-specific packages
pkg/classic/          # Remove
pkg/machinepool/      # Remove (keep nodepool)
pkg/mint/            # Remove

# Modernize remaining packages
pkg/cluster/         # Simplify for HCP only
pkg/nodepool/        # Already HCP-specific
pkg/aws/            # Remove Classic role logic
```

### Phase 3: Modernization (Weeks 5-8)

Replace legacy libraries progressively:
```go
// Old: survey
"github.com/AlecAivazis/survey/v2" → "github.com/charmbracelet/huh"

// Old: color output
"github.com/fatih/color" → "github.com/charmbracelet/lipgloss"

// Old: spinner
"github.com/briandowns/spinner" → "github.com/charmbracelet/bubbles/spinner"
```

### Phase 4: Testing Migration (Weeks 9-10)

1. Remove Classic-specific tests
2. Simplify test scenarios
3. Add HCP-focused integration tests

## File Removal Checklist

### Commands to Remove
- [ ] `cmd/create/machinepool/`
- [ ] `cmd/create/infrastructure/`
- [ ] `cmd/edit/machinepool/`
- [ ] `cmd/describe/machinepool/`
- [ ] `cmd/delete/machinepool/`
- [ ] `cmd/list/infrastructure/`
- [ ] Control plane role commands

### Packages to Remove/Simplify
- [ ] `pkg/aws/policies.go` - Remove Classic policies
- [ ] `pkg/ocm/clusters.go` - Remove Classic cluster logic
- [ ] `pkg/machinepool/` - Remove entirely
- [ ] `tests/e2e/test_rosacli_machine_pool.go` - Remove
- [ ] Classic upgrade logic

### Flags to Remove
- [ ] `--sts/--non-sts` - Always STS for HCP
- [ ] `--mint-mode` - Not supported
- [ ] `--controlplane-iam-role` - Not needed for HCP
- [ ] `--multi-az` at cluster level - Now at nodepool level

## Version Strategy

### v1.x (main branch)
```yaml
version: 1.x.x
support: Classic + HCP
status: Maintenance mode
updates: Security fixes only
EOL: 12 months after v2.0.0 release
```

### v2.x (v2-hcp-only branch)
```yaml
version: 2.x.x
support: HCP only
status: Active development
updates: New features + fixes
target: Default version by Q2 2025
```

## User Communication

### Installation Instructions
```bash
# For Classic clusters (deprecated)
brew install rosa@1

# For HCP clusters (recommended)
brew install rosa  # defaults to v2

# Or specific versions
rosa version 1.x.x  # Classic + HCP
rosa version 2.x.x  # HCP only
```

### Migration Messages
```go
// In v1.x when creating Classic cluster
func createClassicCluster() {
    warning("Classic clusters are deprecated. Consider using HCP clusters.")
    warning("Support will end on DATE. Use rosa v1.x for Classic clusters.")
}

// In v2.x if Classic flag detected
func handleClassicFlag() {
    error("Classic clusters are not supported in rosa v2.x")
    error("Please install rosa v1.x: brew install rosa@1")
}
```

## CI/CD Changes

### GitHub Actions Workflow
```yaml
name: Release
on:
  push:
    tags:
      - 'v1.*'  # Triggers v1 release
      - 'v2.*'  # Triggers v2 release

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Determine version
        id: version
        run: |
          if [[ ${{ github.ref }} == refs/tags/v1.* ]]; then
            echo "branch=main" >> $GITHUB_OUTPUT
          elif [[ ${{ github.ref }} == refs/tags/v2.* ]]; then
            echo "branch=v2-hcp-only" >> $GITHUB_OUTPUT
          fi
      - name: Checkout branch
        run: git checkout ${{ steps.version.outputs.branch }}
      - name: Build and release
        run: make release
```

## Benefits of Branch Approach

1. **Gradual Migration**: Users can migrate at their own pace
2. **Preserved History**: Git blame and history remain useful
3. **Existing Infrastructure**: CI/CD, issues, PRs all continue
4. **Clear Versioning**: Semantic versioning indicates breaking changes
5. **Parallel Support**: Can backport critical fixes to v1.x

## Sample Timeline

```mermaid
gantt
    title ROSA CLI HCP Migration Timeline
    dateFormat  YYYY-MM-DD
    section Preparation
    Branch Setup           :2024-01-01, 1w
    Initial Cleanup        :1w
    section Development
    Remove Classic Code    :2w
    Modernize Libraries    :4w
    Testing & QA          :2w
    section Release
    v2.0.0-beta           :milestone
    v2.0.0 GA             :milestone
    section Maintenance
    v1.x Security Only    :12w
    v1.x EOL              :milestone
```

## Rollback Strategy

If issues arise with v2.x:
1. Users can immediately switch back to v1.x
2. Both versions can coexist on system
3. Clear documentation on version differences

## Success Metrics

- [ ] 50% reduction in codebase size
- [ ] 30% improvement in command execution time
- [ ] 80% test coverage (simplified scenarios)
- [ ] Zero Classic-related code in v2.x
- [ ] Clean separation of concerns

## Decision Points

### When to Remove v1.x Support?
- 12 months after v2.0.0 GA
- When Classic cluster creation is disabled
- When 90% of clusters are HCP

### How to Handle Breaking Changes?
- All breaking changes in v2.0.0
- Minor versions (2.1, 2.2) are backwards compatible
- Clear migration guide for each breaking change
