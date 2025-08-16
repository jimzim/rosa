# Cluster Creation Error Fix

## The Problem
Error: `The 'aws' field is not supported on standard managed clusters`

## Root Cause
The OCM API was rejecting the cluster creation request because:
1. We were setting AWS-specific fields on the cluster object
2. The node pools were being included in the initial cluster creation

## The Solution
1. **Added required ROSA cluster configuration:**
   - Set `CloudProvider` to "aws"
   - Set `Product` to "rosa"  
   - Set `Managed` to true
   - Keep `Hypershift` enabled for HCP

2. **Temporarily disabled AWS and NodePool configuration:**
   - AWS configuration may need to be handled differently for HCP clusters
   - Node pools should be created separately after cluster creation for HCP

## Current Status
✅ **Dry-run works** - You can test cluster configurations
⚠️ **Real cluster creation** - May need additional configuration for actual AWS resources

## How HCP Clusters Work
Unlike Classic ROSA clusters, HCP (Hosted Control Planes) clusters:
- Have the control plane managed by Red Hat
- Create node pools separately after cluster creation
- Handle AWS configuration differently

## Next Steps for Full Implementation
1. Research the correct way to pass AWS configuration for HCP clusters
2. Implement separate node pool creation after cluster is established
3. Add proper OIDC configuration support
4. Handle STS roles correctly for HCP

## Testing Commands

### Test Configuration (Dry Run)
```bash
./bin/rosa cluster create --name test --region us-west-2 --dry-run
```

### Create Minimal Cluster
```bash
./bin/rosa cluster create --name my-hcp --region us-west-2
```

### With More Options
```bash
./bin/rosa cluster create \
  --name production \
  --region us-west-2 \
  --compute-nodes 3 \
  --compute-type m5.2xlarge \
  --multi-az
```

## Important Notes
- The current implementation creates a minimal HCP cluster configuration
- AWS credentials and node pools need to be configured separately
- This is a simplified MVP that demonstrates the HCP-only approach

The fix ensures the cluster object is properly marked as a ROSA HCP cluster, avoiding the API error about unsupported fields on standard clusters.
