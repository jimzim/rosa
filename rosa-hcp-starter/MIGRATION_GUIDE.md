# Migration Guide: Classic ROSA → ROSA HCP

## 🚀 Overview

This guide helps you migrate from Classic ROSA clusters to ROSA HCP (Hosted Control Planes). This new version of the `rosa` CLI is designed exclusively for HCP clusters, which represent the future of OpenShift on AWS. While the command remains `rosa` for familiarity, this version no longer supports Classic clusters.

## ⚠️ Key Differences

### What's Changed

| Aspect | Classic ROSA | ROSA HCP |
|--------|-------------|----------|
| **CLI Support** | `rosa` (original version) | `rosa` (HCP-only version) |
| **Control Plane** | In your AWS account | Red Hat managed |
| **Minimum Nodes** | 6 (3 control + 3 worker) | 2 (workers only) |
| **VPC** | Optional (can create) | **Required** (BYOVPC) |
| **STS** | Optional | **Mandatory** |
| **Provisioning Time** | 30-45 minutes | 10-15 minutes |
| **Machine Pools** | Yes | **NodePools** (different API) |

## 📋 Pre-Migration Checklist

Before migrating to HCP:

- [ ] Existing VPC with subnets in at least 2 AZs
- [ ] AWS account initialized for ROSA
- [ ] STS mode enabled
- [ ] OIDC provider configured
- [ ] Account roles created
- [ ] Operator roles ready

## 🔄 Command Mapping

### Authentication

```bash
# Classic ROSA (original CLI)
rosa login

# HCP-only version (this CLI - browser-based with PKCE)
rosa login --use-auth-code
```

### Cluster Creation

```bash
# Classic
rosa create cluster --cluster-name my-cluster

# HCP (VPC required)
rosa create cluster \
  --cluster-name my-hcp-cluster \
  --subnet-ids subnet-xxx,subnet-yyy,subnet-zzz \
  --sts \
  --mode auto
```

### Node Management

```bash
# Classic (Machine Pools)
rosa create machinepool --cluster my-cluster --name worker --replicas 3

# HCP (NodePools)
rosa create nodepool --cluster my-hcp-cluster --name worker --replicas 3
```

### Scaling

```bash
# Classic
rosa edit machinepool --cluster my-cluster --replicas 5

# HCP
rosa edit nodepool --cluster my-hcp-cluster --nodepool worker --replicas 5
```

### Identity Providers

```bash
# Classic
rosa create idp --cluster my-cluster --type github

# HCP (same command, different backend)
rosa create idp --cluster my-hcp-cluster --type github
```

## 🚦 Step-by-Step Migration

### Step 1: Prepare Infrastructure

```bash
# Create VPC if you don't have one
rosa create network --name production-vpc --region us-east-1

# Or use existing VPC - get subnet IDs
aws ec2 describe-subnets --filters "Name=vpc-id,Values=vpc-xxxxx" \
  --query 'Subnets[].SubnetId' --output text
```

### Step 2: Set Up IAM

```bash
# Create account-wide roles
rosa create account-roles --mode auto --yes

# Create OIDC provider
rosa create oidc-config --mode auto --yes

# Note the OIDC config ID for cluster creation
rosa list oidc-config
```

### Step 3: Create HCP Cluster

```bash
# Interactive mode (easiest)
rosa create cluster --interactive

# Or specify all parameters
rosa create cluster \
  --cluster-name production-hcp \
  --region us-east-1 \
  --version 4.14.latest \
  --subnet-ids subnet-aaa,subnet-bbb,subnet-ccc \
  --compute-nodes 3 \
  --compute-machine-type m5.xlarge \
  --sts \
  --mode auto
```

### Step 4: Create NodePools

```bash
# Default nodepool is created automatically
# Add additional nodepools as needed

rosa create nodepool \
  --cluster production-hcp \
  --name gpu-workers \
  --instance-type p3.2xlarge \
  --replicas 2 \
  --labels workload=ai \
  --taints gpu=true:NoSchedule
```

### Step 5: Configure Access

```bash
# Create admin user
rosa create admin --cluster production-hcp

# Or set up IdP
rosa create idp \
  --cluster production-hcp \
  --type openid \
  --name corporate-sso \
  --client-id rosa \
  --client-secret <secret> \
  --issuer-url https://sso.company.com
```

### Step 6: Migrate Workloads

```bash
# Get credentials for both clusters
oc login <classic-cluster-api>
oc project my-app
oc get all -o yaml > classic-resources.yaml

# Switch to HCP cluster
oc login <hcp-cluster-api>
oc create namespace my-app
oc apply -f classic-resources.yaml
```

## 🆕 New HCP-Only Features

### TuningConfigs (Performance Optimization)

```bash
# Not available in Classic!
rosa create tuning-config \
  --cluster my-hcp \
  --name performance \
  --spec-file tuned.yaml

rosa edit nodepool \
  --cluster my-hcp \
  --nodepool worker \
  --tuning-configs performance
```

### Break-glass Credentials

```bash
# Emergency access (HCP-optimized)
rosa create break-glass-credential \
  --cluster my-hcp \
  --username emergency \
  --expiration 24h
```

### External Authentication

```bash
# Enhanced in HCP
rosa create external-auth-provider \
  --cluster my-hcp \
  --issuer-url https://auth.example.com \
  --client-id my-app
```

## ⚠️ Common Issues & Solutions

### Issue: "AWS field is mandatory"
**Solution**: HCP requires VPC/subnets. Provide `--subnet-ids`.

```bash
rosa create cluster --subnet-ids subnet-xxx,subnet-yyy,subnet-zzz ...
```

### Issue: "STS is required for HCP clusters"
**Solution**: Always use `--sts` flag and ensure roles exist.

```bash
rosa create account-roles --mode auto
rosa create cluster --sts ...
```

### Issue: "Cannot access control plane nodes"
**Solution**: This is by design. Control plane is managed by Red Hat.

```bash
# Use OpenShift APIs instead
oc get nodes
oc debug node/<worker-node>
```

### Issue: "Machine pool commands not working"
**Solution**: Use NodePool commands instead.

```bash
# Replace 'machinepool' with 'nodepool'
rosa create nodepool ...
rosa list nodepools ...
```

## 📊 Feature Comparison

### What's Preserved

✅ Cluster lifecycle (create, delete, upgrade)  
✅ Identity providers  
✅ RBAC and user management  
✅ Add-ons  
✅ Private clusters  
✅ Proxy configuration  
✅ Ingress management  

### What's Different

🔄 Machine Pools → NodePools  
🔄 Control plane access → API-only  
🔄 Cluster creation requires VPC  
🔄 STS mandatory (not optional)  

### What's New

🆕 TuningConfigs  
🆕 Faster provisioning  
🆕 Lower costs (no control plane instances)  
🆕 Enhanced break-glass access  
🆕 Better external auth support  

## 🎯 Best Practices

1. **Use Interactive Mode**: For first-time setup
   ```bash
   rosa create cluster --interactive
   ```

2. **Prepare VPC First**: HCP requires existing VPC
   ```bash
   rosa create network --name my-vpc
   ```

3. **Use NodePools**: More flexible than classic machine pools
   ```bash
   rosa create nodepool --auto-scaling --min 2 --max 10
   ```

4. **Enable Monitoring**: From the start
   ```bash
   rosa create cluster --disable-workload-monitoring=false
   ```

5. **Set Up IDPs Early**: Before granting access
   ```bash
   rosa create idp --type github --organizations my-org
   ```

## 📚 Resources

- [ROSA HCP Documentation](https://docs.openshift.com/rosa/rosa_hcp)
- [HCP Architecture Guide](https://docs.openshift.com/rosa/rosa_architecture/rosa-architecture)
- [AWS VPC Best Practices](https://docs.aws.amazon.com/vpc/latest/userguide/vpc-security-best-practices.html)
- [STS Setup Guide](https://docs.openshift.com/rosa/rosa_install_access_delete_clusters/rosa-sts-creating-a-cluster-quickly)

## 🆘 Getting Help

- **Issues**: [GitHub Issues](https://github.com/openshift/rosa/issues)
- **Support**: Red Hat Customer Portal
- **Community**: [OpenShift Slack](https://openshift.slack.com)

---

**Remember**: HCP is the future of ROSA. While the migration requires some adjustments, the benefits include:
- 🚀 Faster deployment
- 💰 Lower costs
- 🔧 Less maintenance
- 📈 Better scalability
- 🔒 Enhanced security
