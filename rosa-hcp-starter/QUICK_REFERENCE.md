# ROSA HCP CLI - Quick Reference 🚀

> **Essential commands for ROSA HCP (Hosted Control Planes) clusters**

## 🔐 Authentication & Setup

```bash
# Login with browser
rosa-hcp login --use-auth-code

# Check login status
rosa-hcp whoami

# Initialize AWS account
rosa-hcp init

# Logout
rosa-hcp logout
```

## 🏗️ Prerequisites

```bash
# Create account IAM roles
rosa-hcp create account-roles --mode auto --yes

# Create OIDC configuration  
rosa-hcp create oidc-config --mode auto --yes

# List OIDC configs
rosa-hcp list oidc-config

# Create VPC
rosa-hcp create network --name my-vpc --region us-east-1
```

## 🎯 Cluster Operations

### Create
```bash
# Interactive (recommended)
rosa-hcp create cluster --interactive

# Quick create
rosa-hcp create cluster \
  --cluster-name my-cluster \
  --subnet-ids subnet-xxx,subnet-yyy,subnet-zzz \
  --sts --mode auto

# Private cluster
rosa-hcp create cluster \
  --cluster-name private-cluster \
  --subnet-ids subnet-xxx,subnet-yyy \
  --private \
  --sts --mode auto
```

### List & Describe
```bash
# List clusters
rosa-hcp list clusters

# Describe cluster
rosa-hcp describe cluster -c my-cluster

# Get credentials
rosa-hcp create admin -c my-cluster
```

### Edit
```bash
# Scale cluster
rosa-hcp edit cluster -c my-cluster \
  --min-replicas 3 --max-replicas 10

# Make private
rosa-hcp edit cluster -c my-cluster --private

# Add proxy
rosa-hcp edit cluster -c my-cluster \
  --http-proxy-url http://proxy:8080 \
  --https-proxy-url https://proxy:8443
```

### Upgrade
```bash
# List available versions
rosa-hcp list versions

# Schedule upgrade
rosa-hcp upgrade cluster -c my-cluster \
  --version 4.14.10 \
  --schedule-date 2024-01-20 \
  --schedule-time 03:00
```

### Delete
```bash
# Delete cluster
rosa-hcp delete cluster -c my-cluster --yes
```

## 👥 NodePool Management

```bash
# Create nodepool
rosa-hcp create nodepool -c my-cluster \
  --name workers \
  --replicas 3 \
  --instance-type m5.xlarge

# List nodepools
rosa-hcp list nodepools -c my-cluster

# Scale nodepool
rosa-hcp edit nodepool -c my-cluster \
  --nodepool workers \
  --replicas 5

# Add labels and taints
rosa-hcp edit nodepool -c my-cluster \
  --nodepool workers \
  --labels env=prod,team=platform \
  --taints dedicated=backend:NoSchedule

# Delete nodepool
rosa-hcp delete nodepool -c my-cluster \
  --nodepool workers --yes
```

## 🔒 Security & Access

### Admin User
```bash
# Create cluster admin
rosa-hcp create admin -c my-cluster

# Delete admin
rosa-hcp delete admin -c my-cluster --yes
```

### Identity Providers
```bash
# GitHub IDP
rosa-hcp create idp -c my-cluster \
  --type github \
  --name github-auth \
  --organizations my-org

# OIDC IDP
rosa-hcp create idp -c my-cluster \
  --type openid \
  --name corporate \
  --client-id app \
  --client-secret <secret> \
  --issuer-url https://sso.example.com

# List IDPs
rosa-hcp list idps -c my-cluster
```

### User Management
```bash
# Grant cluster-admin
rosa-hcp grant user cluster-admin \
  -c my-cluster \
  --user alice@example.com

# Grant dedicated-admin
rosa-hcp grant user dedicated-admin \
  -c my-cluster \
  --user bob@example.com

# List users
rosa-hcp list users -c my-cluster
```

### Break-glass Access
```bash
# Create emergency credential
rosa-hcp create break-glass-credential \
  -c my-cluster \
  --username emergency \
  --expiration 24h

# List credentials
rosa-hcp list break-glass-credentials -c my-cluster
```

## ⚡ Performance Tuning

### KubeletConfig
```bash
# Create kubelet config
rosa-hcp create kubeletconfig -c my-cluster \
  --name high-pods \
  --pod-pids-limit 4096

# List configs
rosa-hcp list kubeletconfigs -c my-cluster

# Apply to nodepool
rosa-hcp edit nodepool -c my-cluster \
  --nodepool workers \
  --kubelet-configs high-pods
```

### TuningConfig (HCP Exclusive!)
```bash
# Create from file
rosa-hcp create tuning-config -c my-cluster \
  --name performance \
  --spec-file tuned.yaml

# Create example file
rosa-hcp create tuning-config --create-example

# Apply to nodepool
rosa-hcp edit nodepool -c my-cluster \
  --nodepool workers \
  --tuning-configs performance
```

## 🌐 Networking

### Ingress
```bash
# Create additional ingress
rosa-hcp create ingress -c my-cluster \
  --name apps \
  --lb-type nlb \
  --replicas 2

# List ingresses
rosa-hcp list ingresses -c my-cluster

# Edit ingress
rosa-hcp edit ingress -c my-cluster \
  --name apps \
  --replicas 3
```

### DNS Domains
```bash
# Create DNS domain
rosa-hcp create dns-domain --hosted-cp

# List domains
rosa-hcp list dns-domains

# Use in cluster creation
rosa-hcp create cluster \
  --dns-domain-id <domain-id> ...
```

### Network Verification
```bash
# Verify subnets
rosa-hcp verify network \
  --subnet-ids subnet-xxx,subnet-yyy

# Verify from cluster
rosa-hcp verify network -c my-cluster
```

## 📦 Add-ons

```bash
# List available add-ons
rosa-hcp list addons

# Install add-on
rosa-hcp install addon -c my-cluster \
  --addon cluster-logging-operator

# List installed
rosa-hcp list addons -c my-cluster --installed

# Uninstall add-on
rosa-hcp uninstall addon -c my-cluster \
  --addon cluster-logging-operator --yes
```

## 📊 Monitoring & Logs

```bash
# View installation logs
rosa-hcp logs install -c my-cluster

# Watch logs
rosa-hcp logs install -c my-cluster --watch

# Tail logs
rosa-hcp logs install -c my-cluster --tail 100
```

## 🔧 Troubleshooting

```bash
# Check cluster status
rosa-hcp describe cluster -c my-cluster

# Verify permissions
rosa-hcp verify permissions

# List regions
rosa-hcp list regions

# List instance types
rosa-hcp list instance-types

# List versions
rosa-hcp list versions --channel-group stable
```

## 💡 Pro Tips

### Output Formats
```bash
# JSON output
rosa-hcp list clusters -o json

# YAML output  
rosa-hcp describe cluster -c my-cluster -o yaml

# Pipe to jq
rosa-hcp list clusters -o json | jq '.[] | .name'
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
rosa-hcp create cluster --dry-run ...
```

### Interactive Mode
```bash
# Use for complex operations
rosa-hcp create cluster --interactive
rosa-hcp create idp --interactive
```

## 🆘 Common Issues

| Issue | Solution |
|-------|----------|
| "AWS field is mandatory" | Provide `--subnet-ids` |
| "STS is required" | Use `--sts` flag |
| "No OIDC config" | Run `rosa-hcp create oidc-config` |
| "Subnet not found" | Check VPC and region |
| "Permission denied" | Run `rosa-hcp verify permissions` |

## 📚 Help

```bash
# General help
rosa-hcp --help

# Command help
rosa-hcp create cluster --help

# Subcommand help
rosa-hcp create --help
```

---
**Remember**: All commands support `--help` for detailed options!
