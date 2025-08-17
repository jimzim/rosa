# ROSA CLI (HCP Edition) - Quick Reference 🚀

> **Essential commands for ROSA HCP (Hosted Control Planes) clusters**  
> **Note**: This version of `rosa` supports HCP clusters ONLY

## 🔐 Authentication & Setup

```bash
# Login with browser
rosa login --use-auth-code

# Check login status
rosa whoami

# Initialize AWS account
rosa init

# Logout
rosa logout
```

## 🏗️ Prerequisites

```bash
# Create account IAM roles
rosa create account-roles --mode auto --yes

# Create OIDC configuration  
rosa create oidc-config --mode auto --yes

# List OIDC configs
rosa list oidc-config

# Create VPC
rosa create network --name my-vpc --region us-east-1
```

## 🎯 Cluster Operations

### Create
```bash
# Interactive (recommended)
rosa create cluster --interactive

# Quick create
rosa create cluster \
  --cluster-name my-cluster \
  --subnet-ids subnet-xxx,subnet-yyy,subnet-zzz \
  --sts --mode auto

# Private cluster
rosa create cluster \
  --cluster-name private-cluster \
  --subnet-ids subnet-xxx,subnet-yyy \
  --private \
  --sts --mode auto
```

### List & Describe
```bash
# List clusters
rosa list clusters

# Describe cluster
rosa describe cluster -c my-cluster

# Get credentials
rosa create admin -c my-cluster
```

### Edit
```bash
# Scale cluster
rosa edit cluster -c my-cluster \
  --min-replicas 3 --max-replicas 10

# Make private
rosa edit cluster -c my-cluster --private

# Add proxy
rosa edit cluster -c my-cluster \
  --http-proxy-url http://proxy:8080 \
  --https-proxy-url https://proxy:8443
```

### Upgrade
```bash
# List available versions
rosa list versions

# Schedule upgrade
rosa upgrade cluster -c my-cluster \
  --version 4.14.10 \
  --schedule-date 2024-01-20 \
  --schedule-time 03:00
```

### Delete
```bash
# Delete cluster
rosa delete cluster -c my-cluster --yes
```

## 👥 NodePool Management

```bash
# Create nodepool
rosa create nodepool -c my-cluster \
  --name workers \
  --replicas 3 \
  --instance-type m5.xlarge

# List nodepools
rosa list nodepools -c my-cluster

# Scale nodepool
rosa edit nodepool -c my-cluster \
  --nodepool workers \
  --replicas 5

# Add labels and taints
rosa edit nodepool -c my-cluster \
  --nodepool workers \
  --labels env=prod,team=platform \
  --taints dedicated=backend:NoSchedule

# Delete nodepool
rosa delete nodepool -c my-cluster \
  --nodepool workers --yes
```

## 🔒 Security & Access

### Admin User
```bash
# Create cluster admin
rosa create admin -c my-cluster

# Delete admin
rosa delete admin -c my-cluster --yes
```

### Identity Providers
```bash
# GitHub IDP
rosa create idp -c my-cluster \
  --type github \
  --name github-auth \
  --organizations my-org

# OIDC IDP
rosa create idp -c my-cluster \
  --type openid \
  --name corporate \
  --client-id app \
  --client-secret <secret> \
  --issuer-url https://sso.example.com

# List IDPs
rosa list idps -c my-cluster
```

### User Management
```bash
# Grant cluster-admin
rosa grant user cluster-admin \
  -c my-cluster \
  --user alice@example.com

# Grant dedicated-admin
rosa grant user dedicated-admin \
  -c my-cluster \
  --user bob@example.com

# List users
rosa list users -c my-cluster
```

### Break-glass Access
```bash
# Create emergency credential
rosa create break-glass-credential \
  -c my-cluster \
  --username emergency \
  --expiration 24h

# List credentials
rosa list break-glass-credentials -c my-cluster
```

## ⚡ Performance Tuning

### KubeletConfig
```bash
# Create kubelet config
rosa create kubeletconfig -c my-cluster \
  --name high-pods \
  --pod-pids-limit 4096

# List configs
rosa list kubeletconfigs -c my-cluster

# Apply to nodepool
rosa edit nodepool -c my-cluster \
  --nodepool workers \
  --kubelet-configs high-pods
```

### TuningConfig (HCP Exclusive!)
```bash
# Create from file
rosa create tuning-config -c my-cluster \
  --name performance \
  --spec-file tuned.yaml

# Create example file
rosa create tuning-config --create-example

# Apply to nodepool
rosa edit nodepool -c my-cluster \
  --nodepool workers \
  --tuning-configs performance
```

## 🌐 Networking

### Ingress
```bash
# Create additional ingress
rosa create ingress -c my-cluster \
  --name apps \
  --lb-type nlb \
  --replicas 2

# List ingresses
rosa list ingresses -c my-cluster

# Edit ingress
rosa edit ingress -c my-cluster \
  --name apps \
  --replicas 3
```

### DNS Domains
```bash
# Create DNS domain
rosa create dns-domain --hosted-cp

# List domains
rosa list dns-domains

# Use in cluster creation
rosa create cluster \
  --dns-domain-id <domain-id> ...
```

### Network Verification
```bash
# Verify subnets
rosa verify network \
  --subnet-ids subnet-xxx,subnet-yyy

# Verify from cluster
rosa verify network -c my-cluster
```

## 📦 Add-ons

```bash
# List available add-ons
rosa list addons

# Install add-on
rosa install addon -c my-cluster \
  --addon cluster-logging-operator

# List installed
rosa list addons -c my-cluster --installed

# Uninstall add-on
rosa uninstall addon -c my-cluster \
  --addon cluster-logging-operator --yes
```

## 📊 Monitoring & Logs

```bash
# View installation logs
rosa logs install -c my-cluster

# Watch logs
rosa logs install -c my-cluster --watch

# Tail logs
rosa logs install -c my-cluster --tail 100
```

## 🔧 Troubleshooting

```bash
# Check cluster status
rosa describe cluster -c my-cluster

# Verify permissions
rosa verify permissions

# List regions
rosa list regions

# List instance types
rosa list instance-types

# List versions
rosa list versions --channel-group stable
```

## 💡 Pro Tips

### Output Formats
```bash
# JSON output
rosa list clusters -o json

# YAML output  
rosa describe cluster -c my-cluster -o yaml

# Pipe to jq
rosa list clusters -o json | jq '.[] | .name'
```

### Environment Variables
```bash
export ROSA_CLUSTER=my-cluster  # Default cluster
export AWS_REGION=us-east-1      # Default region
export ROSA_OUTPUT=json          # Default output
```

### Dry Run
```bash
# Preview changes without applying
rosa create cluster --dry-run ...
```

### Interactive Mode
```bash
# Use for complex operations
rosa create cluster --interactive
rosa create idp --interactive
```

## 🆘 Common Issues

| Issue | Solution |
|-------|----------|
| "AWS field is mandatory" | Provide `--subnet-ids` |
| "STS is required" | Use `--sts` flag |
| "No OIDC config" | Run `rosa create oidc-config` |
| "Subnet not found" | Check VPC and region |
| "Permission denied" | Run `rosa verify permissions` |

## 📚 Help

```bash
# General help
rosa --help

# Command help
rosa create cluster --help

# Subcommand help
rosa create --help
```

---
**Remember**: All commands support `--help` for detailed options!
