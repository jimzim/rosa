# Missing Features for Complete ROSA HCP CLI

## 🔴 Critical Missing Features (High Priority)

### 1. **Cluster Management**
- **Edit Cluster** (`rosa edit cluster`)
  - Update cluster properties (scaling, networking, etc.)
  - Modify cluster configuration post-creation
  
- **Upgrade Cluster** (`rosa upgrade cluster`)
  - Schedule cluster version upgrades
  - Manage upgrade policies
  - Cancel/rollback upgrades

### 2. **User & Access Management**
- **Admin User** (`rosa create/delete admin`)
  - Create cluster-admin user
  - Critical for initial cluster access
  
- **Identity Providers** (`rosa create/list/delete idp`)
  - GitHub, GitLab, Google, LDAP, OpenID
  - Essential for user authentication
  
- **User Management** (`rosa grant/revoke/list users`)
  - Grant dedicated-admin access
  - Manage user permissions

### 3. **Ingress Management**
- **Ingress Controllers** (`rosa create/edit/delete/list ingress`)
  - Custom domain configuration
  - Load balancer customization
  - Route sharding

### 4. **Cluster Add-ons**
- **Add-on Management** (`rosa list/install/uninstall addon`)
  - Managed services (logging, monitoring, etc.)
  - Operator installations
  - Add-on configurations

## 🟡 Important Missing Features (Medium Priority)

### 5. **Cluster Operations**
- **Autoscaler** (`rosa create/edit/delete autoscaler`)
  - Cluster autoscaling configuration
  - Min/max node limits
  
- **Hibernation** (`rosa hibernate/resume cluster`)
  - Pause/resume clusters to save costs
  
- **Logs** (`rosa logs install/uninstall`)
  - Installation/uninstallation logs
  - Debugging support

### 6. **Advanced Configuration**
- **KubeletConfig** (`rosa create/edit/delete kubeletconfig`)
  - Custom kubelet configurations
  - Pod limits, eviction policies
  
- **TuningConfigs** (`rosa create/edit/delete tuning-configs`)
  - Performance tuning profiles
  - Custom kernel parameters
  
- **External Auth Provider** (`rosa create/delete external-auth-provider`)
  - External OIDC providers
  - Custom authentication flows

### 7. **Version & Region Management**
- **List Versions** (`rosa list versions`)
  - Available OpenShift versions
  - Upgrade paths
  
- **List Regions** (`rosa list regions`)
  - Available AWS regions
  - Region capabilities
  
- **List Instance Types** (`rosa list instance-types`)
  - Available EC2 instance types
  - Instance specifications

## 🟢 Nice-to-Have Features (Low Priority)

### 8. **Verification & Validation**
- **Verify Permissions** (`rosa verify permissions`)
  - Check IAM permissions before cluster creation
  
- **Verify Quota** (`rosa verify quota`)
  - Check AWS service quotas
  
- **Verify Network** (`rosa verify network`)
  - Validate VPC and network configuration

### 9. **Advanced Security**
- **Break-glass Credentials** (`rosa create/list/revoke break-glass-credential`)
  - Emergency access credentials
  
- **Managed Services** (`rosa create/list/delete managed-service`)
  - Service accounts for AWS integrations

### 10. **OCM Integration**
- **OCM Roles** (`rosa create/link/unlink ocm-role`)
  - Organization-level IAM roles
  
- **User Roles** (`rosa create/link/unlink user-role`)
  - User-specific IAM roles

### 11. **DNS Management**
- **DNS Domains** (`rosa create/delete dns-domain`)
  - Custom DNS domain management
  - Route53 integration

### 12. **Support Features**
- **Access Requests** (`rosa list/describe access-request`)
  - Red Hat support access requests
  
- **Download Clients** (`rosa download oc/rosa`)
  - Download OpenShift and ROSA CLI

## 📊 Implementation Priority Matrix

| Priority | Feature | Complexity | Impact | Effort |
|----------|---------|------------|--------|--------|
| **P0** | Admin User | Low | Critical | 1-2 hours |
| **P0** | Edit Cluster | Medium | High | 4-6 hours |
| **P0** | Upgrade Cluster | High | Critical | 6-8 hours |
| **P1** | Identity Providers | Medium | High | 4-6 hours |
| **P1** | Ingress Management | Medium | High | 4-6 hours |
| **P1** | List Versions | Low | High | 2-3 hours |
| **P2** | Add-ons | High | Medium | 8-10 hours |
| **P2** | Autoscaler | Medium | Medium | 4-6 hours |
| **P2** | Hibernation | Medium | Medium | 4-6 hours |
| **P3** | Verification Commands | Low | Low | 2-3 hours |
| **P3** | Advanced Configs | High | Low | 8-10 hours |

## 🚀 Recommended Next Steps

### Phase 1: Essential Access (P0)
```bash
# 1. Admin user creation - Without this, users can't access their clusters
rosa create admin --cluster <name>

# 2. Cluster editing - Basic day-2 operations
rosa edit cluster --name <name> --compute-nodes 5

# 3. Cluster upgrades - Keep clusters secure and updated
rosa upgrade cluster --cluster <name> --version 4.14.5
```

### Phase 2: User Management (P1)
```bash
# Identity providers for team access
rosa create idp --cluster <name> --type github

# Ingress for custom domains
rosa create ingress --cluster <name> --domain example.com

# Version listing for upgrade planning
rosa list versions --channel-group stable
```

### Phase 3: Production Features (P2)
```bash
# Autoscaling for dynamic workloads
rosa create autoscaler --cluster <name> --min 3 --max 10

# Add-ons for additional functionality
rosa install addon --cluster <name> cluster-logging-operator

# Hibernation for cost savings
rosa hibernate cluster --name <name>
```

## 📝 Notes for HCP-Specific Implementation

1. **Machine Pools vs Node Pools**: HCP only uses node pools (already implemented ✅)
2. **Control Plane**: Hosted by Red Hat, no control plane configuration needed
3. **OIDC**: Already implemented, but may need operator role integration
4. **STS-Only**: No mint mode support needed (HCP is STS-only)
5. **Networking**: PrivateLink support may need enhancement

## 💡 Quick Wins (Can implement quickly)

1. **rosa create admin** - Essential, simple to implement (~1-2 hours)
2. **rosa list versions** - Query OCM API for available versions (~2 hours)
3. **rosa list regions** - Query AWS/OCM for available regions (~2 hours)
4. **rosa verify permissions** - Check IAM permissions (~3 hours)

## 🏁 Definition of "Complete"

A production-ready ROSA HCP CLI should have at minimum:
- ✅ Cluster CRUD (already done)
- ✅ Node Pool management (already done)
- ✅ IAM roles setup (already done)
- ✅ Network creation (already done)
- ❌ Admin user creation (P0)
- ❌ Cluster editing/upgrading (P0)
- ❌ Identity providers (P1)
- ❌ Ingress management (P1)
- ❌ Basic add-ons (P2)
- ❌ Autoscaling (P2)

The current implementation covers ~40% of full production features. The most critical gap is **admin user creation** - without it, users cannot access their clusters after creation!
