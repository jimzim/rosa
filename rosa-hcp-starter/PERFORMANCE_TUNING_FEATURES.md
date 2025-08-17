# Performance Tuning Features for ROSA HCP

## Overview
This document describes the KubeletConfig and TuningConfig features implemented for ROSA HCP clusters, providing critical performance tuning capabilities for production workloads.

## KubeletConfig

### What It Does
KubeletConfig allows customization of kubelet settings at the node pool level, currently supporting:
- **Pod PIDs Limit**: Maximum number of process IDs per pod (default: 4096, recommended for high-density: 10000+)

### Key Features
- **Named Configurations**: HCP requires named configs (unlike Classic)
- **No Worker Reboots**: Applied without disrupting running workloads (major advantage over Classic)
- **Per-NodePool**: Different configs for different node pools

### Usage

#### Create a KubeletConfig
```bash
# Create a high-density config
rosa create kubeletconfig \
  --cluster my-cluster \
  --name high-density \
  --pod-pids-limit 10000

# Interactive mode
rosa create kubeletconfig --cluster my-cluster --interactive
```

#### List KubeletConfigs
```bash
rosa list kubeletconfigs --cluster my-cluster
```

#### Delete a KubeletConfig
```bash
rosa delete kubeletconfig high-density --cluster my-cluster
```

#### Use with NodePool
```bash
rosa nodepool create \
  --cluster my-cluster \
  --name dense-workers \
  --kubelet-configs high-density \
  --instance-type m5.2xlarge
```

### Use Cases
- **High Pod Density**: Kubernetes operators, CI/CD runners
- **Container-Heavy Workloads**: Microservices architectures
- **Development Environments**: Many small pods

## TuningConfig

### What It Does
TuningConfig applies TuneD profiles for kernel and OS-level optimizations. This is an **HCP-ONLY** feature not available on Classic clusters.

### Key Features
- **TuneD Integration**: Uses Red Hat's performance tuning daemon
- **Kernel Parameters**: Sysctl settings, scheduler tuning
- **JSON/YAML Support**: Flexible spec format
- **Per-NodePool**: Target specific workload requirements

### Usage

#### Create an Example Spec
```bash
# Generate example spec file
rosa create tuning-config --create-example my-tuning.yaml
```

#### Example Spec (database-tuning.yaml)
```yaml
profile:
- name: database-profile
  data: |
    [main]
    summary=Optimized for database workloads
    include=openshift-node
    
    [sysctl]
    # Memory management
    vm.dirty_ratio="20"
    vm.dirty_background_ratio="5"
    vm.swappiness="10"
    
    # Network optimization
    net.core.somaxconn="4096"
    net.ipv4.tcp_max_syn_backlog="8192"
    net.core.netdev_max_backlog="5000"
    
    # Scheduler tuning
    kernel.sched_migration_cost_ns="5000000"
    kernel.numa_balancing="1"

recommend:
- priority: 20
  profile: database-profile
```

#### Create a TuningConfig
```bash
# From spec file
rosa create tuning-config \
  --cluster my-cluster \
  --name database-tuning \
  --spec-path database-tuning.yaml

# Interactive mode
rosa create tuning-config --cluster my-cluster --interactive
```

#### List TuningConfigs
```bash
# Basic list
rosa list tuning-configs --cluster my-cluster

# Show specs
rosa list tuning-configs --cluster my-cluster --show-spec
```

#### Delete a TuningConfig
```bash
rosa delete tuning-config database-tuning --cluster my-cluster
```

#### Use with NodePool
```bash
rosa nodepool create \
  --cluster my-cluster \
  --name db-workers \
  --tuning-configs database-tuning \
  --instance-type r5.4xlarge
```

### Common Tuning Profiles

#### Database Optimization
```yaml
profile:
- name: database
  data: |
    [sysctl]
    vm.dirty_ratio="20"
    vm.swappiness="10"
    kernel.shmmax="68719476736"
    kernel.shmall="4294967296"
```

#### Cache/Redis Optimization
```yaml
profile:
- name: cache
  data: |
    [sysctl]
    vm.overcommit_memory="1"
    net.core.somaxconn="65535"
    tcp_max_syn_backlog="65535"
```

#### Real-time/Low Latency
```yaml
profile:
- name: realtime
  data: |
    [sysctl]
    kernel.sched_min_granularity_ns="10000000"
    kernel.sched_wakeup_granularity_ns="15000000"
    kernel.sched_rt_runtime_us="950000"
```

## Combining Both Features

For maximum performance optimization, use both features together:

```bash
# Create configs
rosa create kubeletconfig \
  --cluster my-cluster \
  --name high-performance \
  --pod-pids-limit 15000

rosa create tuning-config \
  --cluster my-cluster \
  --name performance-tuning \
  --spec-path performance.yaml

# Create optimized node pool
rosa nodepool create \
  --cluster my-cluster \
  --name optimized-workers \
  --kubelet-configs high-performance \
  --tuning-configs performance-tuning \
  --instance-type m5.8xlarge \
  --min-replicas 3 \
  --max-replicas 10
```

## Benefits Over Classic ROSA

### KubeletConfig
- **No Worker Reboots**: Classic requires rolling reboots
- **Named Configs**: Better organization and reusability
- **Per-NodePool**: Classic has single cluster-wide config

### TuningConfig
- **HCP Exclusive**: Not available on Classic at all
- **Advanced Tuning**: Direct kernel parameter control
- **Workload-Specific**: Different tuning per node pool

## Monitoring and Validation

### Verify KubeletConfig
```bash
# Check pod pids limit on a node
oc debug node/<node-name>
cat /host/etc/kubernetes/kubelet.conf | grep podPidsLimit
```

### Verify TuningConfig
```bash
# Check applied sysctl settings
oc debug node/<node-name>
sysctl vm.dirty_ratio
sysctl vm.swappiness
```

## Best Practices

1. **Test First**: Always test configs in non-production
2. **Monitor Impact**: Watch node metrics after applying
3. **Document Changes**: Keep specs in version control
4. **Start Conservative**: Begin with small changes
5. **Profile Workloads**: Understand your application needs

## Troubleshooting

### KubeletConfig Issues
- **Pods Failing**: PIDs limit too low, increase gradually
- **Resource Pressure**: Monitor node memory/CPU usage

### TuningConfig Issues
- **Spec Validation**: Use `--dry-run` to validate
- **Conflicts**: Check for conflicting profiles
- **Performance Degradation**: Some settings may hurt performance

## Implementation Details

### Architecture
- **Service Layer**: `pkg/kubeletconfig/`, `pkg/tuningconfig/`
- **CLI Commands**: `cmd/rosa/commands/kubeletconfig/`, `cmd/rosa/commands/tuningconfig/`
- **OCM Integration**: Uses OpenShift Cluster Manager API
- **NodePool Integration**: Extended to support both config types

### Requirements
- **HCP Clusters Only**: Both features require Hosted Control Planes
- **ROSA Token**: Valid authentication required
- **Permissions**: Cluster admin or equivalent

## Summary

These performance tuning features provide ROSA HCP users with production-grade optimization capabilities:

- **KubeletConfig**: Essential for high-density container workloads
- **TuningConfig**: Critical for specialized workloads (databases, caches, real-time)
- **HCP Advantages**: No reboots, per-nodepool configs, exclusive features
- **Production Ready**: Battle-tested patterns for common workload types

Together, they enable fine-grained performance optimization that matches or exceeds self-managed Kubernetes capabilities.
