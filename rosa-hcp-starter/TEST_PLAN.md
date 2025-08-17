# Test Plan for ROSA HCP-Only CLI

## 📋 Test Strategy Overview

### Testing Levels
1. **Unit Tests** - Service layer logic
2. **Integration Tests** - API interactions
3. **E2E Tests** - Full command flows
4. **Manual Testing** - UX and interactive modes
5. **Performance Tests** - Speed comparisons

### Current Test Coverage
- ✅ **Compilation** - Builds successfully
- ⚠️ **Unit Tests** - Not yet implemented
- ⚠️ **Integration Tests** - Not yet implemented
- ⚠️ **E2E Tests** - Not yet implemented
- ⏳ **Manual Testing** - Ready to begin

## 🧪 Phase 1: Core Functionality Tests

### 1.1 Authentication & Setup
```bash
# Test browser-based auth
./rosa login --use-auth-code
# Expected: Opens browser, completes PKCE flow

# Test token-based auth
export ROSA_TOKEN="..."
./rosa whoami
# Expected: Shows user info

# Test logout
./rosa logout
# Expected: Clears credentials
```

### 1.2 OIDC Configuration
```bash
# Create managed OIDC config
./rosa create oidc-config --managed --region us-west-2
# Expected: Creates config, shows ID

# List OIDC configs
./rosa list oidc-configs
# Expected: Shows all configs

# Delete OIDC config
./rosa delete oidc-config --oidc-config-id <id>
# Expected: Removes config
```

### 1.3 IAM Roles
```bash
# Create account roles
./rosa create account-roles \
  --prefix my-rosa \
  --region us-west-2 \
  --mode auto
# Expected: Creates all required roles

# List account roles
./rosa list account-roles
# Expected: Shows all roles

# Create operator roles
./rosa create operator-roles \
  --prefix my-rosa \
  --oidc-config-id <id> \
  --installer-role-arn <arn>
# Expected: Creates operator roles
```

## 🧪 Phase 2: Cluster Lifecycle Tests

### 2.1 Cluster Creation Tests

#### Basic Cluster
```bash
./rosa create cluster \
  --name test-hcp-basic \
  --region us-west-2 \
  --subnet-ids subnet-xxx,subnet-yyy \
  --role-arn <installer-role> \
  --support-role-arn <support-role> \
  --worker-role-arn <worker-role> \
  --oidc-config-id <id>
```

#### Private Cluster
```bash
./rosa create cluster \
  --name test-hcp-private \
  --private \
  --subnet-ids <private-subnets>
```

#### With Proxy
```bash
./rosa create cluster \
  --name test-hcp-proxy \
  --http-proxy http://proxy:3128 \
  --https-proxy https://proxy:3128 \
  --no-proxy 10.0.0.0/8
```

#### With External Auth
```bash
./rosa create cluster \
  --name test-hcp-extauth \
  --external-auth-providers-enabled
```

#### With Autoscaling
```bash
./rosa create cluster \
  --name test-hcp-autoscale \
  --enable-autoscaling \
  --min-replicas 2 \
  --max-replicas 10
```

### 2.2 Cluster Operations

#### List Clusters
```bash
./rosa list clusters
./rosa list clusters --output json
./rosa list clusters --output yaml
```

#### Describe Cluster
```bash
./rosa describe cluster --cluster test-hcp
./rosa describe cluster --cluster test-hcp --output json
```

#### Edit Cluster
```bash
# Update display name
./rosa edit cluster --cluster test-hcp \
  --display-name "Production HCP"

# Update scaling
./rosa edit cluster --cluster test-hcp \
  --min-replicas 3 \
  --max-replicas 15

# Update proxy
./rosa edit cluster --cluster test-hcp \
  --http-proxy http://newproxy:3128
```

#### Upgrade Cluster
```bash
# Check available versions
./rosa list versions --cluster test-hcp

# Schedule upgrade
./rosa upgrade cluster --cluster test-hcp \
  --version 4.14.1 \
  --schedule "2024-01-15 02:00"
```

#### Delete Cluster
```bash
./rosa delete cluster --cluster test-hcp --yes
```

## 🧪 Phase 3: NodePool Tests

### 3.1 NodePool CRUD
```bash
# Create nodepool
./rosa create nodepool \
  --cluster test-hcp \
  --name workers \
  --replicas 3 \
  --instance-type m5.2xlarge \
  --labels env=prod,team=platform

# List nodepools
./rosa list nodepools --cluster test-hcp

# Edit nodepool
./rosa edit nodepool \
  --cluster test-hcp \
  --nodepool workers \
  --replicas 5

# Delete nodepool
./rosa delete nodepool \
  --cluster test-hcp \
  --nodepool workers
```

### 3.2 NodePool with Tuning
```bash
# Create kubelet config
./rosa create kubeletconfig \
  --cluster test-hcp \
  --name high-pods \
  --pod-pids-limit 4096

# Create tuning config
./rosa create tuning-config \
  --cluster test-hcp \
  --name performance \
  --spec-file tuning.yaml

# Create nodepool with configs
./rosa create nodepool \
  --cluster test-hcp \
  --name tuned-workers \
  --kubelet-configs high-pods \
  --tuning-configs performance
```

## 🧪 Phase 4: Security & Access Tests

### 4.1 External Authentication
```bash
# Create external auth provider
./rosa create external-auth-provider \
  --cluster test-hcp \
  --name azure-ad \
  --issuer-url https://login.microsoft.com/tenant/v2.0 \
  --client-id <client-id> \
  --client-secret <secret> \
  --claim-mappings username=email,groups=groups

# List providers
./rosa list external-auth-providers --cluster test-hcp

# Describe provider
./rosa describe external-auth-provider \
  --cluster test-hcp \
  --name azure-ad

# Delete provider
./rosa delete external-auth-provider \
  --cluster test-hcp \
  --name azure-ad
```

### 4.2 Break-glass Credentials
```bash
# Create break-glass credential
./rosa create break-glass-credential \
  --cluster test-hcp \
  --username emergency-admin \
  --expiration 24h

# List credentials
./rosa list break-glass-credentials --cluster test-hcp

# Describe credential
./rosa describe break-glass-credential \
  --cluster test-hcp \
  --credential-id <id>

# Revoke all credentials
./rosa revoke break-glass-credential \
  --cluster test-hcp \
  --yes
```

### 4.3 User Management
```bash
# Grant admin access
./rosa grant user \
  --cluster test-hcp \
  --username alice@example.com \
  --role cluster-admin

# List users
./rosa list users --cluster test-hcp

# Revoke access
./rosa revoke user \
  --cluster test-hcp \
  --username alice@example.com \
  --role cluster-admin
```

### 4.4 Identity Providers
```bash
# Create HTPasswd IDP
./rosa create idp \
  --cluster test-hcp \
  --type htpasswd \
  --name local-users

# Create GitHub IDP
./rosa create idp \
  --cluster test-hcp \
  --type github \
  --name github-auth \
  --client-id <id> \
  --client-secret <secret> \
  --organizations myorg

# List IDPs
./rosa list idps --cluster test-hcp

# Delete IDP
./rosa delete idp --cluster test-hcp --name github-auth
```

## 🧪 Phase 5: Networking & Ingress Tests

### 5.1 Ingress Management
```bash
# Create additional ingress
./rosa create ingress \
  --cluster test-hcp \
  --name apps \
  --lb-type nlb \
  --private

# List ingresses
./rosa list ingresses --cluster test-hcp

# Edit ingress
./rosa edit ingress \
  --cluster test-hcp \
  --ingress apps \
  --lb-type alb

# Delete ingress
./rosa delete ingress --cluster test-hcp --ingress apps
```

### 5.2 DNS Domains
```bash
# Create DNS domain
./rosa create dns-domain \
  --cluster test-hcp

# List domains
./rosa list dns-domains

# Describe domain
./rosa describe dns-domain --domain-id <id>

# Delete domain
./rosa delete dns-domain --domain-id <id>
```

## 🧪 Phase 6: Add-ons & Services

### 6.1 Add-on Operations
```bash
# List available add-ons
./rosa list addons

# Install add-on
./rosa install addon \
  --cluster test-hcp \
  --addon-id aws-efs-csi-driver

# List installed add-ons
./rosa list addons --cluster test-hcp --installed

# Uninstall add-on
./rosa uninstall addon \
  --cluster test-hcp \
  --addon-id aws-efs-csi-driver
```

## 🧪 Phase 7: Operational Tests

### 7.1 Logs
```bash
# View installation logs
./rosa logs install --cluster test-hcp

# Watch installation logs
./rosa logs install --cluster test-hcp --watch

# View uninstallation logs
./rosa logs uninstall --cluster test-hcp
```

### 7.2 Verification Commands
```bash
# Verify network configuration
./rosa verify network \
  --region us-west-2 \
  --subnet-ids subnet-xxx,subnet-yyy

# Verify permissions
./rosa verify permissions

# Verify quota
./rosa verify quota --region us-west-2
```

## 🧪 Phase 8: Interactive Mode Tests

### Test Interactive Flows
```bash
# Interactive cluster creation
./rosa create cluster --interactive

# Interactive nodepool creation
./rosa create nodepool --cluster test-hcp --interactive

# Interactive IDP creation
./rosa create idp --cluster test-hcp --interactive
```

### Expected Behaviors
- ✅ Rich TUI with Charm libraries
- ✅ Real-time validation
- ✅ Helpful prompts and defaults
- ✅ Clear error messages
- ✅ Progress indicators

## 🧪 Phase 9: Error Handling Tests

### Invalid Operations
```bash
# Classic cluster attempt (should fail gracefully)
./rosa create cluster --name classic --sts=false
# Expected: Clear error about HCP-only support

# Missing prerequisites
./rosa create cluster --name test
# Expected: Clear error about missing VPC/subnets

# Invalid region
./rosa create cluster --region invalid-region
# Expected: Helpful error with valid regions

# Expired credentials
unset ROSA_TOKEN
./rosa list clusters
# Expected: Prompt to login
```

## 🧪 Phase 10: Performance Tests

### Benchmark Commands
```bash
# Time cluster list
time ./rosa list clusters

# Time large nodepool list
time ./rosa list nodepools --cluster test-hcp

# Compare with original CLI
time rosa list clusters # original
time ./rosa list clusters # new

# Memory usage
/usr/bin/time -l ./rosa list clusters
```

### Expected Performance
- ✅ 30-40% faster startup than original
- ✅ Sub-second response for list operations
- ✅ Smooth interactive mode
- ✅ No memory leaks

## 📊 Test Matrix

| Feature | Unit | Integration | E2E | Manual | Status |
|---------|------|-------------|-----|--------|--------|
| Authentication | 🔄 | 🔄 | 🔄 | ✅ | Ready |
| Cluster CRUD | 🔄 | 🔄 | 🔄 | ✅ | Ready |
| NodePool | 🔄 | 🔄 | 🔄 | ✅ | Ready |
| External Auth | 🔄 | 🔄 | 🔄 | ✅ | Ready |
| Break-glass | 🔄 | 🔄 | 🔄 | ✅ | Ready |
| IDPs | 🔄 | 🔄 | 🔄 | ✅ | Ready |
| Ingress | 🔄 | 🔄 | 🔄 | ✅ | Ready |
| Add-ons | 🔄 | 🔄 | 🔄 | ✅ | Ready |
| User Mgmt | 🔄 | 🔄 | 🔄 | ✅ | Ready |
| Logs | 🔄 | 🔄 | 🔄 | ✅ | Ready |

## 🔄 Automated Test Implementation

### Next Steps for Automation

1. **Unit Tests** (Priority 1)
```go
// Example: pkg/cluster/service_test.go
func TestCreateCluster(t *testing.T) {
    // Mock OCM client
    // Test service logic
    // Verify error handling
}
```

2. **Integration Tests** (Priority 2)
```go
// Example: tests/integration/cluster_test.go
func TestClusterLifecycle(t *testing.T) {
    // Use test OCM environment
    // Create, list, delete cluster
    // Verify API calls
}
```

3. **E2E Tests** (Priority 3)
```bash
# Example: tests/e2e/cluster.sh
./rosa create cluster --dry-run ...
# Verify output
# Check error codes
```

## 📈 Success Criteria

### Phase 1 Complete When:
- ✅ All manual tests pass
- ✅ No panic/crashes
- ✅ Error messages are helpful
- ✅ Interactive mode works smoothly

### Phase 2 Complete When:
- ✅ Unit test coverage > 70%
- ✅ Integration tests for critical paths
- ✅ CI/CD pipeline running tests

### Phase 3 Complete When:
- ✅ E2E test suite complete
- ✅ Performance benchmarks documented
- ✅ Load testing completed
- ✅ Security testing passed

## 🚀 Quick Start Testing

```bash
# Build the CLI
cd rosa-hcp-starter
go build ./cmd/rosa

# Set up test environment
export ROSA_TOKEN="your-token"
export AWS_REGION="us-west-2"

# Run basic smoke test
./test-smoke.sh

# Run full test suite (when implemented)
go test ./...
```

## 📝 Test Reporting

Track results in:
- `TEST_RESULTS.md` - Manual test outcomes
- `coverage.html` - Code coverage report
- `benchmark.txt` - Performance results
- GitHub Issues - Bug reports

## 🐛 Known Issues to Test

1. **SDK Compatibility** - External auth claim mappings
2. **Break-glass** - Only bulk revocation works
3. **Shared VPC** - Tag workaround validation
4. **Large clusters** - Performance with many nodepools

---

**Remember**: Start with manual testing to validate functionality, then automate the repetitive tests.
