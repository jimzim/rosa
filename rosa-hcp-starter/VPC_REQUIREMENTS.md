# ROSA HCP VPC Requirements

## ✅ AWS Field is Now Mandatory

You were right! The AWS field is mandatory for ROSA clusters. I've re-enabled it with proper configuration.

## 📋 Prerequisites for ROSA HCP Clusters

### 1. **VPC Must Exist First**
ROSA HCP clusters require an existing VPC with properly configured subnets. Unlike Classic ROSA, HCP doesn't create its own VPC.

### 2. **Create VPC Using ROSA Helper** (from original ROSA CLI)
```bash
# Using the original ROSA CLI
rosa create network --region us-west-2

# This creates a CloudFormation stack with:
# - VPC with public and private subnets
# - Internet Gateway
# - NAT Gateways
# - Route tables
# - Security groups
```

### 3. **Get Subnet IDs from Your VPC**
```bash
# List your VPC subnets
aws ec2 describe-subnets \
  --filters "Name=vpc-id,Values=vpc-xxxxx" \
  --query 'Subnets[*].[SubnetId,AvailabilityZone,CidrBlock]' \
  --output table
```

## 🚀 Creating HCP Cluster with VPC

### Step 1: Ensure VPC Exists
```bash
# Check if you have a VPC
aws ec2 describe-vpcs --query 'Vpcs[*].[VpcId,CidrBlock,Tags[?Key==`Name`].Value|[0]]' --output table
```

### Step 2: Create Cluster with Subnet IDs
```bash
./bin/rosa cluster create \
  --name my-hcp-cluster \
  --region us-west-2 \
  --subnet-ids subnet-abc123,subnet-def456,subnet-ghi789
```

### Step 3: With Additional AWS Configuration
```bash
./bin/rosa cluster create \
  --name production-hcp \
  --region us-west-2 \
  --subnet-ids subnet-abc123,subnet-def456,subnet-ghi789 \
  --role-arn arn:aws:iam::123456789012:role/ROSA-Installer \
  --support-role-arn arn:aws:iam::123456789012:role/ROSA-Support \
  --worker-iam-role arn:aws:iam::123456789012:role/ROSA-Worker \
  --tags env=production,team=platform
```

## 📝 Key Differences: HCP vs Classic

| Aspect | Classic ROSA | ROSA HCP |
|--------|-------------|----------|
| **VPC Creation** | Can create its own | Requires existing VPC |
| **Control Plane** | In your VPC | In Red Hat's account |
| **Subnets** | Creates if needed | Must provide subnet IDs |
| **Network Stack** | Managed by ROSA | You manage the VPC |

## 🔧 Current Implementation

The cluster service now:
1. **Requires AWS field** - No longer optional
2. **Expects subnet IDs** - Warns if not provided
3. **Configures STS** - If IAM roles are provided
4. **Sets tags** - For resource management

## ⚠️ Important Notes

### Without Subnet IDs
If you try to create a cluster without subnet IDs, you'll see:
- Warning in logs: "No subnet IDs provided - cluster creation may fail"
- Suggestion: "Run 'rosa create network' first to create a VPC"

### Real Cluster Creation
For actual cluster creation (not dry-run), you'll need:
1. Valid AWS credentials
2. Existing VPC with subnets
3. Proper IAM roles (can be created with `rosa create account-roles`)
4. OIDC configuration (can be created with `rosa create oidc-config`)

## 🎯 Complete Example Workflow

```bash
# 1. Create VPC (using original ROSA CLI)
rosa create network --region us-west-2

# 2. Get subnet IDs
SUBNET_IDS=$(aws ec2 describe-subnets \
  --filters "Name=tag:Name,Values=*rosa*" \
  --query 'Subnets[*].SubnetId' \
  --output text | tr '\t' ',')

# 3. Create OIDC config
./bin/rosa create oidc-config --managed

# 4. Create cluster
./bin/rosa cluster create \
  --name my-hcp \
  --region us-west-2 \
  --subnet-ids $SUBNET_IDS \
  --oidc-config-id <oidc-id>

# 5. Create node pool after cluster is ready
./bin/rosa nodepool create \
  --cluster my-hcp \
  --name workers \
  --replicas 3
```

## 📊 Testing

### Dry Run with Subnets
```bash
./bin/rosa cluster create \
  --name test \
  --region us-west-2 \
  --subnet-ids subnet-123,subnet-456 \
  --dry-run
```

### Dry Run without Subnets (will warn)
```bash
./bin/rosa cluster create \
  --name test \
  --region us-west-2 \
  --dry-run
```

The AWS configuration is now properly restored and the cluster creation should work with your existing VPC!
