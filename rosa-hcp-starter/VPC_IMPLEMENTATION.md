# VPC Network Implementation for ROSA HCP

## Overview
Implemented VPC creation, listing, and deletion commands for ROSA HCP clusters. This allows users to create AWS VPCs directly from the ROSA CLI without needing the original ROSA CLI or manual AWS setup.

## Commands Implemented

### 1. Create Network
```bash
rosa create network --name myvpc --region us-west-2
```

Features:
- Creates CloudFormation stack with all required resources
- Configurable availability zones (1-4)
- Custom CIDR support
- Interactive mode for guided setup
- Dry run support

Resources Created:
- VPC with DNS support
- Public and private subnets
- Internet Gateway
- NAT Gateway
- Route tables
- Security groups
- VPC endpoints (S3, EC2, KMS)

### 2. List Networks
```bash
rosa list networks --region us-west-2
```

Features:
- Lists all VPC stacks created by rosa
- Table format display
- Shows subnet counts

### 3. Delete Network
```bash
rosa delete network rosa-vpc-myvpc --region us-west-2
```

Features:
- Safe deletion with confirmation
- Force flag to skip confirmation
- Shows what will be deleted

## Implementation Details

### Package Structure
```
pkg/network/
├── service.go           # Core network service
└── templates/
    └── rosa-quickstart-vpc.yaml  # CloudFormation template

cmd/rosa/commands/network/
├── network.go           # Command group
├── create.go           # Create VPC command
├── list.go             # List VPCs command
└── delete.go           # Delete VPC command
```

### Key Components

1. **Network Service** (`pkg/network/service.go`)
   - CloudFormation stack management
   - VPC validation utilities
   - Subnet discovery helpers

2. **CloudFormation Template**
   - Embedded in binary using Go embed
   - Supports up to 4 availability zones
   - Includes all ROSA HCP required tags
   - Configurable CIDR ranges

3. **Integration Points**
   - Uses AWS SDK v2 for CloudFormation and EC2
   - Integrates with existing output formatting
   - Follows CLI patterns for consistency

## Usage Examples

### Create VPC with Default Settings
```bash
./bin/rosa create network --name test-vpc --region us-west-2
```

### Create VPC with Custom CIDR and AZs
```bash
./bin/rosa create network \
  --name prod-vpc \
  --region us-west-2 \
  --cidr 10.1.0.0/16 \
  --availability-zones us-west-2a,us-west-2b
```

### Interactive Mode
```bash
./bin/rosa create network --interactive
```

### Use with Cluster Creation
After creating a VPC, use the subnet IDs in cluster creation:
```bash
# Get subnet IDs from create output or list command
./bin/rosa cluster create \
  --name mycluster \
  --region us-west-2 \
  --subnet-ids subnet-abc123,subnet-def456
```

## Benefits

1. **Simplified Setup**: No need to manually create VPCs in AWS Console
2. **ROSA-Optimized**: Includes all required tags and configurations
3. **Integrated Experience**: Works seamlessly with cluster creation
4. **Safe Operations**: Confirmation prompts and dry-run support

## Next Steps

1. Add JSON/YAML output formats for automation
2. Support for existing VPC import/validation
3. PrivateLink VPC configuration option
4. Multi-region VPC peering setup
5. Integration with cluster creation for automatic VPC selection

## Testing

Test the implementation:
```bash
# Build
make build

# Test create (dry run)
./bin/rosa create network --name test --region us-west-2 --dry-run

# Create actual VPC
./bin/rosa create network --name test --region us-west-2

# List VPCs
./bin/rosa list networks --region us-west-2

# Delete VPC
./bin/rosa delete network rosa-vpc-test --region us-west-2
```

## Known Limitations

1. CloudFormation stack operations can take 5-10 minutes
2. VPC deletion will fail if resources are still in use
3. No support for existing VPC modification
4. Output formats limited to text (JSON/YAML not yet implemented)
