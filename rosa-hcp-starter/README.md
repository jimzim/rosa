# ROSA CLI - HCP Edition 🚀

> **⚠️ IMPORTANT: This CLI supports ROSA HCP (Hosted Control Planes) clusters ONLY**  
> Classic ROSA clusters are NOT supported in this version.

## 📋 Overview

This is the next-generation ROSA CLI built exclusively for **Hosted Control Planes (HCP)** - the future of Red Hat OpenShift on AWS. HCP provides a more scalable, cost-effective, and manageable approach to running OpenShift clusters with the control plane managed by Red Hat.

### Why HCP-Only?

- **Simplified Architecture**: Control plane runs on Red Hat infrastructure
- **Reduced Costs**: No need for control plane EC2 instances
- **Faster Provisioning**: Clusters ready in ~15 minutes
- **Better Multi-tenancy**: Shared control plane infrastructure
- **Enhanced Security**: Isolated control plane with private endpoint options

## 🎯 Key Features

### Core Cluster Operations
- ✅ Create, list, describe, delete HCP clusters
- ✅ Cluster editing and scaling
- ✅ Cluster upgrades
- ✅ NodePool management (CRUD operations)

### Security & Access
- ✅ External authentication providers (OIDC)
- ✅ Break-glass emergency credentials
- ✅ Identity providers (HTPasswd, GitHub, GitLab, Google, LDAP, OIDC)
- ✅ User management and RBAC
- ✅ Admin user creation

### AWS Integration
- ✅ IAM roles creation (account-roles, operator-roles)
- ✅ OIDC configuration
- ✅ STS authentication
- ✅ VPC and network management
- ✅ Audit log forwarding to CloudWatch

### Performance & Operations
- ✅ KubeletConfig customization
- ✅ TuningConfigs (HCP-exclusive feature!)
- ✅ Installation and uninstallation logs
- ✅ Add-ons management
- ✅ Ingress configuration
- ✅ DNS domain management

## 📦 Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/openshift/rosa.git
cd rosa/rosa-hcp-starter

# Build the CLI
make build

# Install to your PATH
sudo mv bin/rosa /usr/local/bin/rosa-hcp

# Verify installation
rosa-hcp version
```

### Download Pre-built Binary (when available)

```bash
# macOS
curl -LO https://github.com/openshift/rosa/releases/download/vX.Y.Z/rosa-hcp-darwin-amd64
chmod +x rosa-hcp-darwin-amd64
sudo mv rosa-hcp-darwin-amd64 /usr/local/bin/rosa-hcp

# Linux
curl -LO https://github.com/openshift/rosa/releases/download/vX.Y.Z/rosa-hcp-linux-amd64
chmod +x rosa-hcp-linux-amd64
sudo mv rosa-hcp-linux-amd64 /usr/local/bin/rosa-hcp
```

## 🚀 Quick Start

### 1. Initial Setup

```bash
# Login to your Red Hat account
rosa-hcp login --use-auth-code

# Verify AWS credentials
aws sts get-caller-identity

# Initialize your AWS account for ROSA
rosa-hcp init

# Verify permissions
rosa-hcp verify permissions
```

### 2. Create Prerequisites

```bash
# Create account-wide IAM roles
rosa-hcp create account-roles --mode auto --yes

# Create OIDC configuration
rosa-hcp create oidc-config --mode auto --yes

# Create a VPC (or use existing)
rosa-hcp create network --name my-vpc --region us-east-1
```

### 3. Create an HCP Cluster

```bash
# Interactive mode (recommended for first time)
rosa-hcp create cluster --interactive

# Or with explicit parameters
rosa-hcp create cluster \
  --cluster-name my-hcp-cluster \
  --region us-east-1 \
  --subnet-ids subnet-xxx,subnet-yyy,subnet-zzz \
  --sts \
  --mode auto
```

### 4. Create a NodePool

```bash
# Create a nodepool for your workloads
rosa-hcp create nodepool \
  --cluster my-hcp-cluster \
  --name worker \
  --replicas 3 \
  --instance-type m5.xlarge
```

### 5. Access Your Cluster

```bash
# Create an admin user
rosa-hcp create admin --cluster my-hcp-cluster

# Get cluster credentials
rosa-hcp describe cluster --cluster my-hcp-cluster

# Configure kubectl
oc login <api-url> -u cluster-admin -p <password>
```

## 📚 Common Workflows

### Cluster Lifecycle Management

```bash
# List all clusters
rosa-hcp list clusters

# Describe cluster details
rosa-hcp describe cluster --cluster my-cluster

# Edit cluster (scaling, network settings, etc.)
rosa-hcp edit cluster --cluster my-cluster --min-replicas 3 --max-replicas 10

# Upgrade cluster
rosa-hcp upgrade cluster --cluster my-cluster --version 4.14.10

# Delete cluster
rosa-hcp delete cluster --cluster my-cluster --yes
```

### NodePool Management

```bash
# List nodepools
rosa-hcp list nodepools --cluster my-cluster

# Scale a nodepool
rosa-hcp edit nodepool --cluster my-cluster --nodepool worker --replicas 5

# Add labels and taints
rosa-hcp edit nodepool --cluster my-cluster --nodepool worker \
  --labels workload=frontend \
  --taints key1=value1:NoSchedule

# Delete nodepool
rosa-hcp delete nodepool --cluster my-cluster --nodepool worker --yes
```

### Security Configuration

```bash
# Configure external authentication
rosa-hcp create external-auth-provider \
  --cluster my-cluster \
  --issuer-url https://auth.example.com \
  --client-id my-app \
  --client-secret <secret>

# Create break-glass credentials (emergency access)
rosa-hcp create break-glass-credential \
  --cluster my-cluster \
  --username emergency-admin \
  --expiration 24h

# Set up identity provider
rosa-hcp create idp \
  --cluster my-cluster \
  --type github \
  --name github-auth \
  --organizations my-org

# Grant user permissions
rosa-hcp grant user cluster-admin \
  --cluster my-cluster \
  --user alice@example.com
```

### Performance Tuning (HCP Exclusive!)

```bash
# Create custom kubelet configuration
rosa-hcp create kubeletconfig \
  --cluster my-cluster \
  --name high-pods \
  --pod-pids-limit 4096

# Create TuningConfig for performance optimization
rosa-hcp create tuning-config \
  --cluster my-cluster \
  --name my-tuning \
  --spec-file tuned-config.yaml

# Apply configs to nodepool
rosa-hcp edit nodepool \
  --cluster my-cluster \
  --nodepool worker \
  --kubelet-configs high-pods \
  --tuning-configs my-tuning
```

### Add-ons and Services

```bash
# List available add-ons
rosa-hcp list addons

# Install an add-on
rosa-hcp install addon \
  --cluster my-cluster \
  --addon cluster-logging-operator

# Configure multiple ingresses
rosa-hcp create ingress \
  --cluster my-cluster \
  --name apps \
  --lb-type nlb \
  --replicas 2
```

### Monitoring and Troubleshooting

```bash
# View installation logs
rosa-hcp logs install --cluster my-cluster --watch

# Verify network configuration
rosa-hcp verify network --subnet-ids subnet-xxx,subnet-yyy

# List cluster versions
rosa-hcp list versions --channel-group stable

# Check cluster status
rosa-hcp describe cluster --cluster my-cluster
```

## 🏗️ Architecture

### HCP Cluster Structure

```
┌─────────────────────────────────────┐
│      Red Hat Infrastructure         │
│  ┌─────────────────────────────┐   │
│  │   Control Plane (Managed)    │   │
│  │  - API Server                │   │
│  │  - etcd                      │   │
│  │  - Controller Manager        │   │
│  │  - Scheduler                 │   │
│  └─────────────────────────────┘   │
└─────────────────────────────────────┘
              ↕️ Private Link
┌─────────────────────────────────────┐
│        Your AWS Account             │
│  ┌─────────────────────────────┐   │
│  │      VPC                     │   │
│  │  ┌────────────────────┐     │   │
│  │  │   NodePool(s)       │     │   │
│  │  │  - Worker Nodes     │     │   │
│  │  │  - Your Workloads   │     │   │
│  │  └────────────────────┘     │   │
│  └─────────────────────────────┘   │
└─────────────────────────────────────┘
```

## 🔧 Configuration

### Config File Location

```bash
~/.rosa-hcp/config.yaml
```

### Environment Variables

```bash
# API Configuration
export ROSA_API_URL=https://api.openshift.com

# AWS Configuration  
export AWS_REGION=us-east-1
export AWS_PROFILE=my-profile

# Authentication
export ROSA_TOKEN=<your-token>

# Output Format
export ROSA_OUTPUT=json  # or yaml, text
```

### Profiles

```bash
# Create a profile
rosa-hcp config set --profile production --region us-east-1

# Use a profile
rosa-hcp list clusters --profile production
```

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## 📝 Differences from Classic ROSA

| Feature | Classic ROSA | ROSA HCP |
|---------|-------------|----------|
| Control Plane Location | Customer AWS Account | Red Hat Infrastructure |
| Control Plane Nodes | 3+ EC2 instances | Shared, managed |
| Provisioning Time | 30-45 minutes | 10-15 minutes |
| Minimum EC2 Instances | 6+ (3 control + 3 worker) | 2+ (workers only) |
| Control Plane Scaling | Manual | Automatic |
| Control Plane Costs | Customer pays | Included in subscription |
| TuningConfigs Support | ❌ | ✅ |
| Multi-zone Control Plane | Requires 3 zones | Not required |
| Private Link Support | ✅ | ✅ |
| BYOVPC | ✅ | ✅ (Required) |

## ⚠️ Important Notes

1. **HCP-Only**: This CLI does NOT support classic ROSA clusters
2. **VPC Required**: HCP clusters require an existing VPC with subnets
3. **STS Mandatory**: All HCP clusters use STS authentication
4. **No Control Plane Access**: Control plane runs on Red Hat infrastructure
5. **NodePools**: Worker nodes are managed through NodePool resources

## 🆘 Support

- **Documentation**: [https://docs.openshift.com/rosa](https://docs.openshift.com/rosa)
- **Issues**: [GitHub Issues](https://github.com/openshift/rosa/issues)
- **Red Hat Support**: Available with active subscription

## 📜 License

Apache License 2.0 - See [LICENSE](LICENSE) for details.

## 🎯 Roadmap

- [x] Core HCP cluster operations
- [x] Complete IAM/STS integration
- [x] NodePool management
- [x] External authentication
- [x] Performance tuning features
- [x] Add-ons support
- [ ] Cluster templates
- [ ] Cost estimation
- [ ] Backup/restore operations
- [ ] GitOps integration

---

**Built with ❤️ for the OpenShift community**