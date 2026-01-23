# LVM Collector

The LVM collector monitors Logical Volume Manager (LVM) storage metrics and automatically restores PVC (PersistentVolumeClaim) sizes when discrepancies are detected between Kubernetes and the underlying LVM volumes.

## Configuration

### YAML Configuration

```yaml
collectors:
  lvm:
    updateInterval: "10s"
    nodeName: ""
```

### Configuration Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `updateInterval` | duration | `10s` | Interval between LVM metrics updates |
| `nodeName` | string | `""` | Node name to monitor (typically set via downward API) |

### Environment Variables

All configuration can be overridden using environment variables with the prefix `COLLECTORS_LVM_`:

| Environment Variable | Maps To | Example |
|---------------------|---------|---------|
| `COLLECTORS_LVM_UPDATE_INTERVAL` | `updateInterval` | `30s` |
| `NODE_NAME` | `nodeName` | `worker-1` |

## Metrics

### `sealos_lvm_vgs_total_capacity`

**Type:** Gauge
**Labels:**
- `node`: Node name

**Description:** Total capacity of all LVM volume groups in bytes.

**Example:**
```promql
sealos_lvm_vgs_total_capacity{node="worker-1"} 1099511627776
sealos_lvm_vgs_total_capacity{node="worker-2"} 2199023255552
```

### `sealos_lvm_vgs_total_free`

**Type:** Gauge
**Labels:**
- `node`: Node name

**Description:** Total free space of all LVM volume groups in bytes.

**Example:**
```promql
sealos_lvm_vgs_total_free{node="worker-1"} 549755813888
sealos_lvm_vgs_total_free{node="worker-2"} 1099511627776
```

## Use Cases

### Monitoring Storage Capacity

```promql
# Calculate storage utilization percentage
(1 - (sealos_lvm_vgs_total_free / sealos_lvm_vgs_total_capacity)) * 100

# Alert when storage usage exceeds 80%
((sealos_lvm_vgs_total_capacity - sealos_lvm_vgs_total_free) / sealos_lvm_vgs_total_capacity) > 0.8

# Alert when free space is below 100GB
sealos_lvm_vgs_total_free < 107374182400
```

### Tracking Storage Growth

```promql
# Storage usage growth rate (bytes per second)
rate(sealos_lvm_vgs_total_free[5m])

# Total storage used across all nodes
sum(sealos_lvm_vgs_total_capacity - sealos_lvm_vgs_total_free)

# Available storage by node
sealos_lvm_vgs_total_free
```

## Deployment Requirements

### Privileged Container

The LVM collector requires privileged access to interact with LVM:

```yaml
securityContext:
  privileged: true
```

### Host Mounts

The following host paths must be mounted:

```yaml
volumeMounts:
  - name: dev
    mountPath: /dev
  - name: lvm-config
    mountPath: /etc/lvm
  - name: kubelet
    mountPath: /var/lib/kubelet
    mountPropagation: Bidirectional
```

### Deployment Mode

The LVM collector must be deployed as a **DaemonSet** to monitor LVM on each node:

```yaml
daemonsetConfig:
  enabled: true
  enabledCollectors:
    - lvm
```

## Collector Type

**Type:** Polling
**Leader Election Required:** No
**Deployment Mode:** DaemonSet

The LVM collector runs as a DaemonSet with one instance per node. Each instance monitors the local LVM volumes and manages PVC restoration for volumes on its node.

## Dependencies

The collector requires the following LVM tools to be installed in the container:

- `lvm2`: Core LVM utilities
- `lvm2-extra`: Additional LVM tools
- `util-linux`: Linux utilities
- `device-mapper`: Device mapper support
- `btrfs-progs`, `xfsprogs`, `e2fsprogs`: Filesystem tools
