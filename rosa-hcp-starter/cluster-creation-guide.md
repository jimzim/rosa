# 🚀 Cluster Creation Guide

## ✅ Your cluster creation is now working!

### Quick Test (Dry Run - No AWS resources created)
```bash
./bin/rosa cluster create --name test-hcp --region us-west-2 --dry-run
```

### Create a Real HCP Cluster

#### Option 1: Basic Creation
```bash
./bin/rosa cluster create \
  --name my-hcp-cluster \
  --region us-west-2
```

#### Option 2: Interactive Mode (Guided)
```bash
./bin/rosa cluster create --interactive
```
This will prompt you for:
- Cluster name
- AWS region
- Multi-AZ configuration
- Private cluster options
- Compute node configuration

#### Option 3: Advanced Configuration
```bash
./bin/rosa cluster create \
  --name production-hcp \
  --region us-west-2 \
  --compute-nodes 3 \
  --compute-type m5.2xlarge \
  --multi-az \
  --version 4.14.0
```

### Available Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--name` | Cluster name (required) | - |
| `--region` | AWS region (required) | - |
| `--compute-nodes` | Number of worker nodes | 2 |
| `--compute-type` | EC2 instance type | m5.xlarge |
| `--multi-az` | Deploy across multiple AZs | true |
| `--private` | Create private cluster | false |
| `--version` | OpenShift version | latest |
| `--dry-run` | Simulate without creating | false |
| `--interactive` | Interactive mode | false |

### What Happens When You Create a Cluster

1. **Validates** your AWS credentials and permissions
2. **Creates** the hosted control plane in Red Hat's AWS account
3. **Sets up** networking and security groups
4. **Deploys** worker nodes in your AWS account
5. **Configures** the OpenShift cluster

### Example: Create Your First HCP Cluster

```bash
# 1. First, make sure you're logged in
./bin/rosa login --use-auth-code

# 2. Create the cluster
./bin/rosa cluster create \
  --name my-first-hcp \
  --region us-west-2

# 3. Check cluster status
./bin/rosa cluster describe my-first-hcp

# 4. List all clusters
./bin/rosa cluster list
```

### Notes

- HCP clusters have the control plane hosted by Red Hat
- You only pay for worker nodes in your AWS account
- No need for `--hosted-cp` flag (all clusters are HCP)
- Cluster creation takes about 15-20 minutes

Ready to create your cluster! 🎉
