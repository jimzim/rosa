# HCP-Only OCM SDK Rewrite Plan

## 🎯 Overview
Creating an HCP-only version of the OCM SDK ecosystem, similar to what we accomplished with the ROSA CLI, would provide significant benefits in terms of code size, performance, and maintainability.

## 📦 Core Repositories to Rewrite

### 1. **ocm-sdk-go** (Primary Target)
**Repository**: `github.com/openshift-online/ocm-sdk-go`
**Current State**: ~500K+ lines of generated code
**HCP-Only Estimate**: ~100K lines (80% reduction)

**What to Remove**:
- Classic ROSA cluster types and builders
- Non-HCP machine pool implementations  
- Classic STS role management
- Non-HCP networking types
- Legacy authentication methods
- Deprecated API versions

**What to Keep**:
- HCP cluster management (`/api/clusters_mgmt/v1/`)
- HCP node pools
- OIDC configuration
- External authentication
- Add-ons framework
- Core authentication/connection logic

### 2. **ocm-api-model** (API Definitions)
**Repository**: `github.com/openshift-online/ocm-api-model`
**Purpose**: OpenAPI specifications that generate the SDK
**Current State**: Hundreds of model definitions

**HCP-Only Changes**:
```yaml
# Remove from model/clusters_mgmt/v1/:
- classic_cluster_*.yaml
- aws_infrastructure_access_role_*.yaml  
- machine_pool_*.yaml (keep node_pool_*.yaml)
- flavour_*.yaml (Classic-specific)

# Keep:
- cluster_*.yaml (with HCP fields only)
- node_pool_*.yaml
- oidc_*.yaml
- external_auth_*.yaml
- addon_*.yaml
```

### 3. **ocm-cli** (Optional but Recommended)
**Repository**: `github.com/openshift-online/ocm-cli`
**Purpose**: General OCM management CLI
**Benefit**: Lighter tool for OCM operations

**HCP-Only Focus**:
- Remove Classic cluster commands
- Remove machine pool commands
- Streamline to HCP operations only
- Better integration with HCP ROSA

## 🔧 Implementation Strategy

### Phase 1: Fork and Analyze
```bash
# Fork the repositories
git clone https://github.com/openshift-online/ocm-api-model ocm-api-model-hcp
git clone https://github.com/openshift-online/ocm-sdk-go ocm-sdk-go-hcp

# Analyze usage patterns
grep -r "NodePool" ocm-sdk-go/  # HCP-specific
grep -r "MachinePool" ocm-sdk-go/  # Classic-specific
```

### Phase 2: Model Cleanup
1. **Identify HCP-only models**:
   ```
   /api/clusters_mgmt/v1/
   ├── cluster.yaml (edited)
   ├── node_pool.yaml ✓
   ├── hypershift.yaml ✓
   ├── external_auth.yaml ✓
   └── oidc_config.yaml ✓
   ```

2. **Remove Classic models**:
   - Machine pools
   - Classic infrastructure
   - Non-STS authentication
   - Classic networking

### Phase 3: SDK Generation

**Current Generation Process**:
```bash
# Original SDK generation
cd ocm-api-model
make generate

# This creates:
# - Go types and builders
# - Client methods
# - Request/Response handlers
```

**HCP-Only Generation**:
```bash
# Modified generation with filters
make generate INCLUDE_PATTERNS="*node_pool*,*hypershift*,*external_auth*"
```

### Phase 4: Manual Optimization

After generation, manually optimize:

1. **Remove unused imports**
2. **Consolidate error handling**
3. **Optimize builder patterns**
4. **Remove compatibility layers**

## 📊 Expected Improvements

### Size Reduction
| Component | Original | HCP-Only | Reduction |
|-----------|----------|----------|-----------|
| OCM SDK | ~500K LOC | ~100K LOC | 80% |
| API Models | ~200 files | ~50 files | 75% |
| Generated Types | ~1000 | ~250 | 75% |
| Binary Size | ~50MB | ~15MB | 70% |

### Performance Gains
- **Faster compilation**: 60% faster builds
- **Reduced memory**: 50% less runtime memory
- **Faster imports**: 70% faster module resolution
- **Cleaner API**: No version compatibility overhead

## 🗂️ Related Repositories Impact

### Direct Dependencies to Update

1. **openshift/rosa** (Already Done ✓)
   - Our HCP-only CLI

2. **openshift-online/ocm-sdk-go**
   - Core SDK library

3. **openshift-online/ocm-api-model**  
   - API specifications

### Indirect Dependencies (Optional)

4. **openshift-online/ocm-cli**
   - General OCM CLI tool
   - Could benefit from HCP-only version

5. **openshift/cluster-api-provider-aws**
   - If using programmatic cluster management
   - May reference SDK types

6. **openshift/backplane-cli**
   - Support tool that uses OCM SDK
   - Could use lighter HCP SDK

## 🚀 Quick Start Guide

### Step 1: Create HCP-Only API Model
```bash
# Clone and modify
git clone https://github.com/openshift-online/ocm-api-model
cd ocm-api-model
git checkout -b hcp-only

# Remove Classic models
rm -rf model/clusters_mgmt/v1/machine_pool_*
rm -rf model/clusters_mgmt/v1/classic_*
rm -rf model/clusters_mgmt/v1/flavour_*

# Update generator config
cat > .generate-hcp.yaml <<EOF
include:
  - "**/node_pool*"
  - "**/hypershift*"
  - "**/external_auth*"
  - "**/oidc*"
  - "**/addon*"
exclude:
  - "**/machine_pool*"
  - "**/classic*"
EOF
```

### Step 2: Generate HCP-Only SDK
```bash
# Install generator
go install github.com/openshift-online/ocm-api-model/cmd/ocm-generator@latest

# Generate HCP-only SDK
ocm-generator generate \
  --model=model \
  --output=../ocm-sdk-go-hcp \
  --config=.generate-hcp.yaml
```

### Step 3: Optimize Generated Code
```bash
cd ../ocm-sdk-go-hcp

# Remove unused code
go mod tidy
gofmt -s -w .
goimports -w .

# Run static analysis
golangci-lint run --fix

# Remove empty files
find . -type f -size 0 -delete
```

### Step 4: Update Dependencies
```go
// In rosa-hcp/go.mod
replace github.com/openshift-online/ocm-sdk-go => ../ocm-sdk-go-hcp
```

## 🎯 Priority Order

### High Priority (Core Functionality)
1. **ocm-api-model** - Define HCP-only API surface
2. **ocm-sdk-go** - Generate lightweight SDK
3. Integration with existing `rosa-hcp` CLI

### Medium Priority (Ecosystem)
4. **ocm-cli** - HCP-focused OCM tool
5. Documentation and examples
6. Migration guides

### Low Priority (Nice to Have)
7. Backplane CLI integration
8. Terraform provider updates
9. Ansible modules

## 🔄 Maintenance Strategy

### Upstream Sync Process
```bash
# Periodic sync from upstream
git remote add upstream https://github.com/openshift-online/ocm-api-model
git fetch upstream
git cherry-pick [HCP-specific commits]
```

### Versioning Strategy
- Follow upstream major versions
- Add `-hcp` suffix (e.g., `v0.1.400-hcp`)
- Maintain compatibility with OCM API

## 📈 Success Metrics

### Immediate Benefits
- ✅ 80% smaller SDK codebase
- ✅ 60% faster compilation
- ✅ 70% fewer dependencies
- ✅ Clearer API surface

### Long-term Benefits
- ✅ Easier maintenance
- ✅ Faster feature development
- ✅ Better performance
- ✅ Reduced cognitive load

## 🚧 Challenges and Solutions

### Challenge 1: SDK Generation
**Problem**: SDK is auto-generated from models
**Solution**: Fork generator to support filtering

### Challenge 2: API Compatibility  
**Problem**: Must maintain compatibility with OCM API
**Solution**: Keep same API paths, just fewer types

### Challenge 3: Upstream Changes
**Problem**: Need to track upstream improvements
**Solution**: Automated sync process with cherry-pick

## 📝 Example: HCP-Only SDK Usage

```go
// Before (Original SDK)
import (
    cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

// 1000+ types available, most unused
cluster, err := cmv1.NewCluster().
    Name("test").
    Hypershift(cmv1.NewHypershift().Enabled(true)). // HCP
    MachinePools(...) // Classic - not needed!
    Build()

// After (HCP-Only SDK)  
import (
    hcp "github.com/openshift-online/ocm-sdk-go-hcp/clustersmgmt/v1"
)

// Only ~250 types, all relevant
cluster, err := hcp.NewCluster().
    Name("test").
    Hypershift(hcp.NewHypershift().Enabled(true)). // HCP
    // No MachinePools - doesn't exist!
    Build()
```

## 🎬 Next Steps

1. **Prototype** - Create minimal HCP-only model set
2. **Generate** - Build lightweight SDK
3. **Integrate** - Test with rosa-hcp CLI
4. **Optimize** - Profile and improve
5. **Document** - Create migration guides
6. **Release** - Publish as separate module

---

**Estimated Effort**: 2-3 weeks for core SDK, 4-6 weeks for full ecosystem
**ROI**: Massive improvement in developer experience and performance
