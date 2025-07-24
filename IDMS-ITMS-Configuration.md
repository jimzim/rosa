# IDMS and ITMS Configuration for ROSA HCP

This document describes how to configure Image Digest Mirror Sets (IDMS) and Image Tag Mirror Sets (ITMS) for ROSA HCP clusters to support image mirroring in disconnected environments.

## Overview

IDMS and ITMS are OpenShift resources that configure the container runtime to redirect image pulls from one registry to another. This is essential for disconnected or air-gapped environments where direct access to public registries is not available.

- **IDMS (Image Digest Mirror Sets)**: Redirect image pulls by digest
- **ITMS (Image Tag Mirror Sets)**: Redirect image pulls by tag

## Configuration Format

Both IDMS and ITMS use the same JSON configuration format:

```json
[
  {
    "source": "registry.redhat.io",
    "mirrors": [
      "mirror.registry.local/redhat",
      "backup.registry.local/redhat"
    ]
  },
  {
    "source": "quay.io/openshift",
    "mirrors": [
      "mirror.registry.local/openshift"
    ]
  }
]
```

Each entry contains:
- `source`: The original registry to redirect from
- `mirrors`: Array of mirror registries to redirect to (in order of preference)

## Creating a Cluster with IDMS/ITMS

### Using CLI Flags

```bash
# Create a ROSA HCP cluster with IDMS configuration
rosa create cluster \
  --cluster-name my-hcp-cluster \
  --hosted-cp \
  --registry-config-image-digest-mirror-sets /path/to/idms.json \
  --registry-config-image-tag-mirror-sets /path/to/itms.json \
  --region us-east-1

# Or just with IDMS
rosa create cluster \
  --cluster-name my-hcp-cluster \
  --hosted-cp \
  --registry-config-image-digest-mirror-sets /path/to/idms.json \
  --region us-east-1
```

### Interactive Mode

When using `--interactive` mode, you will be prompted to configure image mirroring settings if you enable registry configuration.

## Editing Existing Clusters

```bash
# Update IDMS/ITMS configuration for an existing HCP cluster
rosa edit cluster my-hcp-cluster \
  --registry-config-image-digest-mirror-sets /path/to/updated-idms.json \
  --registry-config-image-tag-mirror-sets /path/to/updated-itms.json
```

## Example Configuration Files

### IDMS Configuration (idms.json)

```json
[
  {
    "source": "registry.redhat.io",
    "mirrors": [
      "mirror.registry.local:5000/redhat",
      "backup.mirror.local:5000/redhat"
    ]
  },
  {
    "source": "quay.io/openshift-release-dev",
    "mirrors": [
      "mirror.registry.local:5000/openshift-release-dev"
    ]
  },
  {
    "source": "quay.io/openshift",
    "mirrors": [
      "mirror.registry.local:5000/openshift"
    ]
  }
]
```

### ITMS Configuration (itms.json)

```json
[
  {
    "source": "docker.io/library",
    "mirrors": [
      "mirror.registry.local:5000/library"
    ]
  },
  {
    "source": "gcr.io",
    "mirrors": [
      "mirror.registry.local:5000/gcr"
    ]
  }
]
```

## Best Practices

1. **Test Configuration**: Validate your mirror registries are accessible before applying the configuration
2. **Multiple Mirrors**: Specify multiple mirror registries for redundancy
3. **Order Matters**: List mirrors in order of preference
4. **Complete Coverage**: Ensure all required registries for your workloads are covered
5. **Monitor**: Check cluster events and pod status after applying mirror configuration

## Validation

The ROSA CLI performs validation on IDMS/ITMS configurations:

- Source registry cannot be empty
- At least one mirror must be specified
- Mirror registries cannot be empty
- JSON format must be valid

## Troubleshooting

If image pulls fail after configuring IDMS/ITMS:

1. Verify mirror registries are accessible from cluster nodes
2. Check that all required images are available in mirror registries
3. Validate JSON configuration syntax
4. Review cluster events for image pull errors
5. Ensure mirror registry authentication is properly configured

## Related Commands

```bash
# View current cluster registry configuration
rosa describe cluster my-hcp-cluster

# List all HCP clusters
rosa list clusters --hosted-cp

# Check cluster status
rosa describe cluster my-hcp-cluster --output json
```

## Limitations

- IDMS and ITMS configuration is only supported for ROSA HCP (Hosted Control Plane) clusters
- Changes to image mirroring configuration may require pod restarts
- Mirror registries must contain all required images and tags