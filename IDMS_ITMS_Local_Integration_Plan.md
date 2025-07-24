# 🚀 **Local Integration Plan: IDMS/ITMS Across 3 Repositories**

## 📋 **Overview**

This plan enables local integration testing of IDMS/ITMS support across three repositories on the `feat/IDMS-ITMS-support` branch, allowing others to download and test the complete implementation.

## 📊 **Current Repository Status**

| Repository | Local Path | Branch | Current Status |
|------------|------------|---------|----------------|
| **ocm-api-model** | `/Users/jzimmerm/projects/redhat/ocm-api-model` | `feat/IDMS-ITMS-support` | Needs IDMS/ITMS types |
| **ocm-sdk-go** | `/Users/jzimmerm/projects/redhat/ocm-sdk-go` | `feat/IDMS-ITMS-support` | Uses `ocm-api-model v0.0.428` |
| **rosa** | `/Users/jzimmerm/projects/redhat/rosa` | `feat/IDMS-ITMS-support` | ✅ Implementation Complete |

---

## 🏗️ **Phase 1: Update OCM API Model**

### **Step 1.1: Add IDMS/ITMS Types to API Model**

Create new model files in `../ocm-api-model/model/clusters_mgmt/v1/`:

**File: `image_digest_mirror_set_type.model`**
```model
/*
Copyright (c) 2024 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// ImageDigestMirrorSet represents a set of mirrors for digest-based image mirroring.
// Each ImageDigestMirrorSet contains a list of mirrors that can serve images
// based on digest matching. This is the modern replacement for ImageContentSourcePolicy (ICSP)
// starting with OpenShift 4.13+.
struct ImageDigestMirrorSet {
	// Name is a human-readable name for the mirror set
	Name String
	// Mirrors contains the mapping from a source registry to one or more mirror registries
	Mirrors []ImageMirror
}
```

**File: `image_tag_mirror_set_type.model`**
```model
/*
Copyright (c) 2024 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// ImageTagMirrorSet represents a set of mirrors for tag-based image mirroring.
// Each ImageTagMirrorSet contains a list of mirrors that can serve images
// based on tag matching. This is the modern replacement for ImageContentSourcePolicy (ICSP)
// starting with OpenShift 4.13+.
struct ImageTagMirrorSet {
	// Name is a human-readable name for the mirror set
	Name String
	// Mirrors contains the mapping from a source registry to one or more mirror registries
	Mirrors []ImageMirror
}
```

**File: `image_mirror_type.model`**
```model
/*
Copyright (c) 2024 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// ImageMirror represents a mapping from a source registry to one or more mirror registries.
// It supports both digest-based and tag-based mirroring depending on the context.
struct ImageMirror {
	// Source is the registry location that images are pulled from
	Source String
	// MirrorsByDigest contains one or more alternative locations that can serve the same images by digest
	MirrorsByDigest []String
	// MirrorsByTag contains one or more alternative locations that can serve the same images by tag
	MirrorsByTag []String
}
```

### **Step 1.2: Update ClusterRegistryConfig**

Update `../ocm-api-model/model/clusters_mgmt/v1/cluster_registry_config_type.model`:

Add these fields to `ClusterRegistryConfig` struct after the `RegistrySources` field:

```model
	// ImageDigestMirrorSets contains the configuration for IDMS (ImageDigestMirrorSet)
	// which provides digest-based image mirroring functionality for images from non-RH registries.
	// This is the modern replacement for ImageContentSourcePolicy (ICSP) starting with OpenShift 4.13+.
	ImageDigestMirrorSets []ImageDigestMirrorSet
	// ImageTagMirrorSets contains the configuration for ITMS (ImageTagMirrorSet)  
	// which provides tag-based image mirroring functionality for images from non-RH registries.
	// This is the modern replacement for ImageContentSourcePolicy (ICSP) starting with OpenShift 4.13+.
	ImageTagMirrorSets []ImageTagMirrorSet
```

---

## 🔄 **Phase 2: Regenerate OCM SDK**

### **Step 2.1: Update OCM SDK Branch**

```bash
cd /Users/jzimmerm/projects/redhat/ocm-sdk-go

# Ensure you're on the right branch
git checkout feat/IDMS-ITMS-support 2>/dev/null || git checkout -b feat/IDMS-ITMS-support

# Update go.mod to use your local API model
go mod edit -replace github.com/openshift-online/ocm-api-model/model=../ocm-api-model/model
go mod tidy
```

### **Step 2.2: Regenerate SDK Code**

```bash
cd /Users/jzimmerm/projects/redhat/ocm-api-model

# Generate the new API model code
make generate

cd /Users/jzimmerm/projects/redhat/ocm-sdk-go

# Regenerate SDK with new types
make generate
```

### **Step 2.3: Verify OCM SDK Compilation**

```bash
cd /Users/jzimmerm/projects/redhat/ocm-sdk-go

# Test compilation
make examples

# Verify new types exist
grep -r "ImageDigestMirrorSet\|ImageTagMirrorSet" clustersmgmt/v1/
```

---

## 🔗 **Phase 3: Update ROSA CLI Integration**

### **Step 3.1: Update ROSA to Use Local OCM SDK**

```bash
cd /Users/jzimmerm/projects/redhat/rosa

# Update go.mod to use your local OCM SDK  
go mod edit -replace github.com/openshift-online/ocm-sdk-go=../ocm-sdk-go
go mod tidy
go mod vendor
```

### **Step 3.2: Complete OCM Integration in ROSA**

Update `pkg/ocm/registry_config.go` to remove TODOs and implement actual IDMS/ITMS building:

```go
// In BuildRegistryConfig() function, replace the TODO section with:

if len(spec.ImageDigestMirrorSets) > 0 {
    for _, idms := range spec.ImageDigestMirrorSets {
        idmsBuilder := cmv1.NewImageDigestMirrorSet().Name(idms.Name)
        
        for _, mirror := range idms.Mirrors {
            mirrorBuilder := cmv1.NewImageMirror().
                Source(mirror.Source)
            
            if len(mirror.MirrorsByDigest) > 0 {
                mirrorBuilder = mirrorBuilder.MirrorsByDigest(mirror.MirrorsByDigest...)
            }
            
            idmsBuilder = idmsBuilder.Mirrors(mirrorBuilder)
        }
        
        clusterRegistryConfig = clusterRegistryConfig.ImageDigestMirrorSets(idmsBuilder)
    }
}

if len(spec.ImageTagMirrorSets) > 0 {
    for _, itms := range spec.ImageTagMirrorSets {
        itmsBuilder := cmv1.NewImageTagMirrorSet().Name(itms.Name)
        
        for _, mirror := range itms.Mirrors {
            mirrorBuilder := cmv1.NewImageMirror().
                Source(mirror.Source)
            
            if len(mirror.MirrorsByTag) > 0 {
                mirrorBuilder = mirrorBuilder.MirrorsByTag(mirror.MirrorsByTag...)
            }
            
            itmsBuilder = itmsBuilder.Mirrors(mirrorBuilder)
        }
        
        clusterRegistryConfig = clusterRegistryConfig.ImageTagMirrorSets(itmsBuilder)
    }
}
```

### **Step 3.3: Update Describe Functionality**

Update `cmd/describe/cluster/cmd.go` to display IDMS/ITMS information:

```go
// Replace the TODO with actual display logic:

if cluster.RegistryConfig() != nil {
    registryConfig := cluster.RegistryConfig()
    
    // Display IDMS
    if idmsList, ok := registryConfig.GetImageDigestMirrorSets(); ok && len(idmsList) > 0 {
        writer.String("Image Digest Mirror Sets:\n")
        for _, idms := range idmsList {
            writer.String(fmt.Sprintf("  - Name: %s\n", idms.Name()))
            if mirrors, ok := idms.GetMirrors(); ok {
                for _, mirror := range mirrors {
                    writer.String(fmt.Sprintf("    Source: %s\n", mirror.Source()))
                    if digestMirrors, ok := mirror.GetMirrorsByDigest(); ok {
                        writer.String(fmt.Sprintf("    Mirrors by Digest: %v\n", digestMirrors))
                    }
                }
            }
        }
    }
    
    // Display ITMS  
    if itmsList, ok := registryConfig.GetImageTagMirrorSets(); ok && len(itmsList) > 0 {
        writer.String("Image Tag Mirror Sets:\n")
        for _, itms := range itmsList {
            writer.String(fmt.Sprintf("  - Name: %s\n", itms.Name()))
            if mirrors, ok := itms.GetMirrors(); ok {
                for _, mirror := range mirrors {
                    writer.String(fmt.Sprintf("    Source: %s\n", mirror.Source()))
                    if tagMirrors, ok := mirror.GetMirrorsByTag(); ok {
                        writer.String(fmt.Sprintf("    Mirrors by Tag: %v\n", tagMirrors))
                    }
                }
            }
        }
    }
}
```

---

## ✅ **Phase 4: Local Integration Testing**

### **Step 4.1: Comprehensive Compilation Test**

```bash
# Test all three repositories compile
cd /Users/jzimmerm/projects/redhat/ocm-api-model && make generate
cd /Users/jzimmerm/projects/redhat/ocm-sdk-go && make generate && make examples  
cd /Users/jzimmerm/projects/redhat/rosa && make rosa
```

### **Step 4.2: Local CLI Testing**

```bash
cd /Users/jzimmerm/projects/redhat/rosa

# Test IDMS flag
./rosa create cluster --help | grep -A5 -B5 "image-digest-mirror-sets"

# Test ITMS flag  
./rosa create cluster --help | grep -A5 -B5 "image-tag-mirror-sets"

# Test validation
./rosa create cluster test-cluster \
  --registry-config-image-digest-mirror-sets="test-idms:registry.example.com=mirror1.io,mirror2.io" \
  --dry-run
```

### **Step 4.3: Unit Test Verification**

```bash
cd /Users/jzimmerm/projects/redhat/rosa

# Run existing validation tests
go test ./pkg/clusterregistryconfig/... -v

# Run OCM integration tests  
go test ./pkg/ocm/... -v
```

---

## 🚀 **Phase 5: Prepare for Sharing**

### **Step 5.1: Clean Up Dependencies for Sharing**

```bash
# In ocm-sdk-go, prepare for sharing (remove local replace)
cd /Users/jzimmerm/projects/redhat/ocm-sdk-go
go mod edit -dropreplace github.com/openshift-online/ocm-api-model/model

# Update to point to your fork instead
go mod edit -require github.com/YOUR_GITHUB_USERNAME/ocm-api-model/model@feat-IDMS-ITMS-support

# In rosa, update to point to your SDK fork
cd /Users/jzimmerm/projects/redhat/rosa
go mod edit -dropreplace github.com/openshift-online/ocm-sdk-go
go mod edit -require github.com/YOUR_GITHUB_USERNAME/ocm-sdk-go@feat-IDMS-ITMS-support
```

### **Step 5.2: Commit and Push All Branches**

```bash
# Push API Model changes
cd /Users/jzimmerm/projects/redhat/ocm-api-model
git add .
git commit -m "feat: Add IDMS/ITMS types to cluster registry config

- Add ImageDigestMirrorSet, ImageTagMirrorSet, and ImageMirror types
- Update ClusterRegistryConfig to include IDMS/ITMS fields  
- Modern replacement for ImageContentSourcePolicy (ICSP) for OpenShift 4.13+"
git push origin feat/IDMS-ITMS-support

# Push OCM SDK changes
cd /Users/jzimmerm/projects/redhat/ocm-sdk-go
git add .
git commit -m "feat: Generate SDK with IDMS/ITMS support

- Regenerated from ocm-api-model with IDMS/ITMS types
- Includes ImageDigestMirrorSet and ImageTagMirrorSet builders"
git push origin feat/IDMS-ITMS-support

# Push ROSA CLI changes  
cd /Users/jzimmerm/projects/redhat/rosa
git add .
git commit -m "feat: Complete IDMS/ITMS integration with OCM SDK

- Remove TODO comments and implement actual OCM integration
- Add display functionality for describe cluster command
- Update dependencies to use feat/IDMS-ITMS-support branches"
git push origin feat/IDMS-ITMS-support
```

---

## 🎯 **Success Criteria**

✅ **All three repositories compile without errors**  
✅ **ROSA CLI accepts IDMS/ITMS flags**  
✅ **OCM SDK contains new types**  
✅ **Integration tests pass**  
✅ **Others can clone and test your branches**

---

## 🚨 **Next Steps After Implementation**

1. **Share Repository URLs** with your team
2. **Create Pull Requests** in each upstream repository  
3. **Run E2E Tests** against actual OCM backend
4. **Gather Feedback** from other developers
5. **Iterate** based on testing results

---

## 📚 **Related Documentation**

- [IDMS/ITMS CLI Implementation Plan](./IDMS_ITMS_CLI_Implementation_Plan.md) - Original CLI-only implementation
- [Integration Test Guide](./IDMS_ITMS_Integration_Test_Guide.md) - Step-by-step testing instructions
- [OpenShift Image Configuration Docs](https://docs.openshift.com/container-platform/4.16/openshift_images/image-configuration.html)

---

This plan ensures a complete, testable integration across all three repositories while maintaining clean, shareable branches for collaboration. 