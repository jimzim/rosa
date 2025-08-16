# ROSA CLI HCP-Only Rewrite Plan

## Executive Summary

This document outlines a comprehensive strategy for creating a new version of the ROSA CLI that exclusively supports ROSA HCP (Hosted Control Planes), dropping support for ROSA Classic as it will be deprecated. This rewrite represents an opportunity to modernize the codebase, improve maintainability, and optimize specifically for HCP workflows.

## Current State Analysis

### Architecture Overview
- **Language**: Go 1.23.1
- **CLI Framework**: Cobra (spf13/cobra)
- **API Client**: OCM SDK Go (openshift-online/ocm-sdk-go)
- **AWS SDK**: AWS SDK Go v2
- **Testing**: Ginkgo/Gomega for e2e tests, standard Go testing for unit tests

### HCP vs Classic Differences

#### Core Differences
1. **Control Plane Management**
   - HCP: Control plane runs in Red Hat's AWS account (multi-tenant)
   - Classic: Control plane runs in customer's account

2. **Node Management**
   - HCP: Uses NodePools (more flexible, version-specific)
   - Classic: Uses MachinePools

3. **IAM Roles**
   - HCP: Only 3 account roles (Installer, Support, Worker)
   - Classic: 4 account roles (includes Control Plane role)

4. **Features Exclusive to HCP**
   - Tuning Configs (kernel tuning for nodes)
   - Shared VPC with specific endpoint roles
   - NodePool-specific upgrades
   - Billing account integration for AWS Marketplace
   - No CNI option (bring your own CNI)

5. **Features Not Available in HCP**
   - Infrastructure nodes
   - Control plane role management
   - Multi-AZ at cluster level (handled at NodePool level)

## Proposed Architecture

### 1. Project Structure
```
rosa-hcp/
├── cmd/
│   └── rosa/
│       ├── main.go
│       └── commands/
│           ├── cluster/
│           ├── nodepool/
│           ├── account/
│           ├── network/
│           └── auth/
├── pkg/
│   ├── api/
│   │   ├── client/
│   │   └── models/
│   ├── aws/
│   │   ├── sts/
│   │   ├── vpc/
│   │   └── iam/
│   ├── cluster/
│   ├── nodepool/
│   ├── upgrade/
│   └── utils/
├── internal/
│   ├── config/
│   ├── output/
│   └── validation/
└── tests/
    ├── unit/
    ├── integration/
    └── e2e/
```

### 2. Core Design Principles

#### 2.1 HCP-First Design
- Remove all conditional logic checking for HCP vs Classic
- Design APIs and workflows optimized for HCP patterns
- Simplify command structure by removing Classic-specific flags

#### 2.2 Modern Go Patterns
- Use Go 1.23 features (improved error handling, generics where appropriate)
- Context-aware operations throughout
- Structured logging with slog
- Proper dependency injection

#### 2.3 Improved User Experience
- Better interactive mode with modern prompts
- Rich terminal output with progress indicators
- Comprehensive validation with helpful error messages
- Smart defaults based on HCP best practices

## Migration Strategy

### Phase 1: Foundation (Weeks 1-4)

#### 1.1 Core Infrastructure
- [ ] Set up new repository structure
- [ ] Implement core command framework (Cobra setup)
- [ ] Create base client for OCM API
- [ ] Implement authentication/configuration management
- [ ] Set up structured logging
- [ ] Create output formatting framework (JSON, YAML, table)

#### 1.2 AWS Integration
- [ ] Implement AWS client wrapper with proper error handling
- [ ] Create STS role management for HCP (3 roles only)
- [ ] Implement OIDC provider management
- [ ] Create VPC/networking utilities for HCP shared VPC

### Phase 2: Cluster Management (Weeks 5-8)

#### 2.1 Cluster Operations
```go
// Simplified cluster creation for HCP
type CreateClusterOptions struct {
    Name                    string
    Region                  string
    Version                 string
    AWS                     AWSConfig
    Network                 NetworkConfig
    NodePools               []NodePoolConfig
    BillingAccount          string
    ExternalAuthentication  bool
}
```

- [ ] `create cluster` - HCP-optimized workflow
- [ ] `delete cluster` 
- [ ] `describe cluster`
- [ ] `list clusters`
- [ ] `edit cluster` - limited to HCP-editable fields
- [ ] `hibernate/resume cluster`

#### 2.2 NodePool Management (HCP-specific)
- [ ] `create nodepool`
- [ ] `delete nodepool`
- [ ] `edit nodepool`
- [ ] `list nodepools`
- [ ] `describe nodepool`

### Phase 3: Advanced Features (Weeks 9-12)

#### 3.1 HCP-Specific Features
- [ ] Tuning configs management
- [ ] External authentication providers
- [ ] Shared VPC configuration
- [ ] Break-glass credentials
- [ ] Audit log forwarding
- [ ] Managed services integration

#### 3.2 Upgrade Management
- [ ] Control plane upgrades (automatic/manual)
- [ ] NodePool upgrades (independent versioning)
- [ ] Upgrade policies

### Phase 4: Testing & Documentation (Weeks 13-16)

#### 4.1 Testing Strategy
```go
// Example of HCP-focused testing
func TestCreateHCPCluster(t *testing.T) {
    // No need for Classic/HCP branching
    cluster := NewHCPCluster(opts)
    // Direct HCP testing
}
```

- [ ] Unit tests (80%+ coverage)
- [ ] Integration tests with mock OCM
- [ ] E2E tests for critical paths
- [ ] Performance benchmarks

#### 4.2 Documentation
- [ ] Command reference documentation
- [ ] HCP migration guide from Classic
- [ ] API documentation
- [ ] Interactive tutorials

## Key Improvements Over Current Implementation

### 1. Removed Complexity
- No Classic/HCP branching logic
- No deprecated features
- Simplified IAM role management (3 vs 4 roles)
- No mint mode support (STS only)

### 2. HCP Optimizations
```go
// Before (current)
if cluster.Hypershift().Enabled() {
    // HCP logic
} else {
    // Classic logic
}

// After (new)
// Direct HCP implementation
cluster.CreateNodePool(opts)
```

### 3. Better Command Structure
```bash
# Current (mixed Classic/HCP)
rosa create cluster --hosted-cp
rosa create machinepool  # or nodepool for HCP

# New (HCP-only)
rosa create cluster  # Always HCP
rosa create nodepool # Clear naming
```

### 4. Improved Error Handling
```go
type ROSAError struct {
    Code        string
    Message     string
    Details     map[string]interface{}
    Suggestion  string
}
```

## Command Mapping

### Core Commands to Implement

| Category | Commands | HCP-Specific Changes |
|----------|----------|---------------------|
| **Cluster** | create, delete, describe, list, edit, hibernate, resume | Simplified flags, no Classic options |
| **NodePool** | create, delete, describe, list, edit, upgrade | Replaces machinepool completely |
| **Account** | create account-roles, create oidc-provider | Only 3 roles, simplified flow |
| **Network** | create vpc-endpoint, verify network | HCP shared VPC specific |
| **Upgrade** | upgrade cluster, upgrade nodepool | Separate control plane/nodepool |
| **Auth** | create idp, create external-auth-provider | HCP-specific auth flows |
| **Tuning** | create tuning-config, list, describe, delete | HCP-only feature |

### Commands to Remove
- Machine pool commands (replaced by nodepool)
- Infrastructure node commands (not supported in HCP)
- Control plane role management (not needed for HCP)
- Mint mode/non-STS operations

## Technical Decisions

### 1. Dependencies
- **Keep**: OCM SDK, AWS SDK v2, Cobra, survey for interactive
- **Replace**: Consider replacing current error handling with more structured approach
- **Add**: Modern testing tools, better observability

### 2. Configuration
```yaml
# Simplified config for HCP
version: 2
profiles:
  default:
    region: us-west-2
    sts:
      role_arn: arn:aws:iam::123456789012:role/ManagedOpenShift-HCP-ROSA-Installer-Role
      support_role_arn: arn:aws:iam::123456789012:role/ManagedOpenShift-HCP-ROSA-Support-Role
      worker_role_arn: arn:aws:iam::123456789012:role/ManagedOpenShift-HCP-ROSA-Worker-Role
```

### 3. API Client Layer
```go
// Clean HCP-focused client
type HCPClient struct {
    ocm *ocmsdk.Connection
    aws *awsclient.Client
}

func (c *HCPClient) CreateCluster(opts CreateClusterOptions) (*Cluster, error) {
    // Direct HCP cluster creation
    // No Classic branching
}
```

## Success Metrics

### Performance
- 50% reduction in binary size (removing Classic code)
- 30% faster command execution (no conditional logic)
- Improved startup time

### Code Quality
- 80%+ test coverage
- Zero Classic-related code
- Consistent error handling
- Clear separation of concerns

### User Experience
- Simplified command structure
- Better error messages with actionable suggestions
- Faster interactive mode
- Consistent output formatting

## Risk Mitigation

### 1. OCM API Changes
- Abstract OCM client interface for easier updates
- Version the client to handle API changes
- Comprehensive integration tests

### 2. Feature Parity
- Ensure all HCP features are covered
- Migration guide for Classic users
- Clear documentation of differences

### 3. Adoption
- Provide migration tools/scripts
- Maintain compatibility with existing OIDC configs
- Clear versioning strategy

## Implementation Timeline

### Month 1: Foundation
- Core structure and basic commands
- Authentication and configuration
- AWS integration

### Month 2: Cluster Operations
- Full cluster lifecycle management
- NodePool management
- Basic upgrade support

### Month 3: Advanced Features
- Tuning configs
- External auth
- Shared VPC
- Complete upgrade management

### Month 4: Polish & Release
- Comprehensive testing
- Documentation
- Performance optimization
- Beta release

## Maintenance Strategy

### Versioning
- Semantic versioning (v2.0.0 for HCP-only)
- Clear deprecation policy
- Regular release cycle (monthly)

### Support
- HCP-only support going forward
- Classic users directed to v1.x (current version)
- Clear EOL timeline for Classic support

### Future Enhancements
- GitOps integration
- Advanced cost management
- Cluster templates
- Policy-based management

## Conclusion

This rewrite presents an opportunity to create a cleaner, more maintainable, and more performant ROSA CLI specifically optimized for HCP clusters. By removing Classic support, we can:

1. Reduce codebase complexity by ~40%
2. Improve user experience with HCP-focused workflows
3. Accelerate feature development for HCP
4. Provide better performance and reliability

The proposed architecture and implementation plan ensures a smooth transition while delivering a superior tool for managing ROSA HCP clusters.
