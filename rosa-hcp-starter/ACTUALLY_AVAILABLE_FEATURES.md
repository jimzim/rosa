# Actually Available Features from Existing ROSA CLI

After analyzing the existing ROSA CLI codebase, here's what's ACTUALLY available and could be added to our HCP implementation:

## ✅ Features That ARE Implemented & HCP-Compatible

### 1. **Ingress Management** - FULLY AVAILABLE
```bash
rosa create ingress  # Not yet implemented but supported in existing CLI
rosa edit ingress
rosa delete ingress  
rosa list ingresses
rosa describe ingress
```
- Custom domains, load balancer types (Classic/NLB)
- Route selectors, wildcard policies
- Private ingress support
- **Implementation Effort**: ~8-10 hours

### 2. **Add-ons & Managed Services** - FULLY AVAILABLE
```bash
rosa install addon
rosa uninstall addon
rosa list addons
```
- Full add-on lifecycle management
- Billing model support
- STS role creation for add-ons
- **Implementation Effort**: ~10-12 hours

### 3. **Instance Types Listing** - FULLY AVAILABLE
```bash
rosa list instance-types
```
- Region-specific instance types
- HCP-compatible instances
- **Implementation Effort**: ~3-4 hours

### 4. **KubeletConfig** - HCP SUPPORTED
```bash
rosa create kubeletconfig
rosa edit kubeletconfig
rosa delete kubeletconfig
rosa list kubeletconfigs
```
- The code explicitly handles HCP clusters (line 86-96 in cmd/create/kubeletconfig/cmd.go)
- Supports pod-pids-limit configuration
- **Implementation Effort**: ~6-8 hours

### 5. **TuningConfigs** - HCP ONLY!
```bash
rosa create tuning-config
rosa delete tuning-config
rosa list tuning-configs
```
- Actually REQUIRES HCP (CheckIfHypershiftClusterOrExit)
- Custom kernel tuning
- **Implementation Effort**: ~6-8 hours

### 6. **Break-glass Credentials** - HCP SUPPORTED
```bash
rosa create break-glass-credential
rosa list break-glass-credentials
rosa revoke break-glass-credential
```
- Emergency access support
- Kubeconfig retrieval
- **Implementation Effort**: ~6-8 hours

### 7. **List Regions** - FULLY AVAILABLE
```bash
rosa list regions
rosa list rh-regions
```
- Available AWS regions
- HCP-enabled regions
- **Implementation Effort**: ~2-3 hours

### 8. **External Auth Providers** - HCP SUPPORTED
```bash
rosa create external-auth-provider
rosa delete external-auth-provider
rosa list external-auth-providers
```
- External OIDC providers
- **Implementation Effort**: ~6-8 hours

### 9. **DNS Domains** - FULLY AVAILABLE
```bash
rosa create dns-domain
rosa delete dns-domain
rosa list dns-domains
```
- Custom DNS domain management
- **Implementation Effort**: ~4-6 hours

### 10. **Verification Commands** - FULLY AVAILABLE
```bash
rosa verify permissions
rosa verify quota
rosa verify network
```
- Pre-flight checks
- **Implementation Effort**: ~4-6 hours

## ❌ Features NOT Available for HCP

### 1. **Autoscaler** - NOT SUPPORTED
- Code explicitly checks and returns error for HCP clusters:
  ```go
  if cluster.Hypershift().Enabled() {
      return fmt.Errorf("Hosted Control Plane clusters do not support cluster-autoscaler configuration")
  }
  ```

### 2. **Hibernation** - Technical Preview / Hidden
- Marked as `Hidden: true` in command
- Technical Preview with limited support
- You mentioned it's not supported for HCP yet

### 3. **Some IdP Types** - Need Investigation
- LDAP and generic OpenID might be available
- Need to check OCM API support for HCP

## 📊 Revised Implementation Priority

Based on what's ACTUALLY available:

| Feature | HCP Support | Effort | Priority |
|---------|------------|--------|----------|
| **List instance-types** | ✅ Yes | 3 hrs | P0 - Quick win |
| **List regions** | ✅ Yes | 2 hrs | P0 - Quick win |
| **Ingress** | ✅ Yes | 8-10 hrs | P1 - High value |
| **Add-ons** | ✅ Yes | 10-12 hrs | P1 - High value |
| **Verification** | ✅ Yes | 4-6 hrs | P1 - Better UX |
| **KubeletConfig** | ✅ Yes | 6-8 hrs | P2 - Advanced |
| **TuningConfigs** | ✅ Yes (HCP only!) | 6-8 hrs | P2 - Advanced |
| **Break-glass** | ✅ Yes | 6-8 hrs | P2 - Emergency |
| **External Auth** | ✅ Yes | 6-8 hrs | P2 - Enterprise |
| **DNS Domains** | ✅ Yes | 4-6 hrs | P3 - Optional |
| ~~Autoscaler~~ | ❌ No | N/A | Not possible |
| ~~Hibernation~~ | ❌ No | N/A | Not supported |

## 🚀 Quick Wins (Can implement in ~5 hours)

1. **List instance-types** (~3 hours)
   - Essential for choosing node pool instance types
   - Simple API call wrapper

2. **List regions** (~2 hours)  
   - Shows available regions for cluster creation
   - Simple listing command

## 🎯 High-Impact Next Features (~30 hours total)

1. **Ingress Management** (~10 hours)
   - Critical for production applications
   - Custom domains are essential

2. **Add-ons** (~12 hours)
   - Logging, monitoring, service mesh
   - Highly requested by users

3. **Verification Commands** (~6 hours)
   - Better user experience
   - Catch errors before cluster creation

## 💡 The Real Gap

The actual gap is smaller than initially thought:
- **Original estimate**: Missing 30% of features
- **Reality**: Missing ~15-20% of features
- Many "missing" features ARE available for HCP

Most importantly, features like:
- TuningConfigs are HCP-ONLY (we should definitely add this!)
- KubeletConfig DOES support HCP
- Break-glass credentials work with HCP
- External auth providers work with HCP

## 📈 Revised Coverage

With the 5 critical features already implemented:
- **Current**: ~70% coverage
- **With quick wins** (instance-types, regions): ~75% coverage  
- **With high-impact** (ingress, add-ons, verify): ~85% coverage
- **With all available HCP features**: ~90-95% coverage

The remaining 5-10% are features that either:
- Don't support HCP (autoscaler, hibernation)
- Are rarely used (some OCM role management)
- Are enterprise-specific edge cases
