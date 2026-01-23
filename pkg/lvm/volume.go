package lvm

import "k8s.io/apimachinery/pkg/api/resource"

// lvm vg, lv & pv fields related constants
const (
	VGName              = "vg_name"
	VGUUID              = "vg_uuid"
	VGPVvount           = "pv_count"
	VGLvCount           = "lv_count"
	VGMaxLv             = "max_lv"
	VGMaxPv             = "max_pv"
	VGSnapCount         = "snap_count"
	VGMissingPvCount    = "vg_missing_pv_count"
	VGMetadataCount     = "vg_mda_count"
	VGMetadataUsedCount = "vg_mda_used_count"
	VGSize              = "vg_size"
	VGFreeSize          = "vg_free"
	VGMetadataSize      = "vg_mda_size"
	VGMetadataFreeSize  = "vg_mda_free"
	VGPermissions       = "vg_permissions"
	VGAllocationPolicy  = "vg_allocation_policy"

	LVName            = "lv_name"
	LVFullName        = "lv_full_name"
	LVUUID            = "lv_uuid"
	LVPath            = "lv_path"
	LVDmPath          = "lv_dm_path"
	LVActive          = "lv_active"
	LVSize            = "lv_size"
	LVMetadataSize    = "lv_metadata_size"
	LVSegtype         = "segtype"
	LVHost            = "lv_host"
	LVPool            = "pool_lv"
	LVPermissions     = "lv_permissions"
	LVWhenFull        = "lv_when_full"
	LVHealthStatus    = "lv_health_status"
	RaidSyncAction    = "raid_sync_action"
	LVDataPercent     = "data_percent"
	LVMetadataPercent = "metadata_percent"
	LVSnapPercent     = "snap_percent"

	PVName             = "pv_name"
	PVUUID             = "pv_uuid"
	PVInUse            = "pv_in_use"
	PVAllocatable      = "pv_allocatable"
	PVMissing          = "pv_missing"
	PVSize             = "pv_size"
	PVFreeSize         = "pv_free"
	PVUsedSize         = "pv_used"
	PVMetadataSize     = "pv_mda_size"
	PVMetadataFreeSize = "pv_mda_free"
	PVDeviceSize       = "dev_size"
)

var Enums = map[string][]string{
	"lv_permissions":       {"unknown", "writeable", "read-only", "read-only-override"},
	"lv_when_full":         {"error", "queue"},
	"raid_sync_action":     {"idle", "frozen", "resync", "recover", "check", "repair"},
	"lv_health_status":     {"", "partial", "refresh needed", "mismatches exist"},
	"vg_allocation_policy": {"normal", "contiguous", "cling", "anywhere", "inherited"},
	"vg_permissions":       {"writeable", "read-only"},
}

// VolumeGroup specifies attributes of a given vg exists on node.
type VolumeGroup struct {
	// Name of the lvm volume group.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// UUID denotes a unique identity of a lvm volume group.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	UUID string `json:"uuid"`

	// Size specifies the total size of volume group.
	// +kubebuilder:validation:Required
	Size resource.Quantity `json:"size"`
	// Free specifies the available capacity of volume group.
	// +kubebuilder:validation:Required
	Free resource.Quantity `json:"free"`

	// LVCount denotes total number of logical volumes in
	// volume group.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=0
	LVCount int32 `json:"lvCount"`
	// PVCount denotes total number of physical volumes
	// constituting the volume group.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=0
	PVCount int32 `json:"pvCount"`

	// MaxLV denotes maximum number of logical volumes allowed
	// in volume group or 0 if unlimited.
	MaxLV int32 `json:"maxLv"`

	// MaxPV denotes maximum number of physical volumes allowed
	// in volume group or 0 if unlimited.
	MaxPV int32 `json:"maxPv"`

	// SnapCount denotes number of snapshots in volume group.
	SnapCount int32 `json:"snapCount"`

	// MissingPVCount denotes number of physical volumes in
	// volume group which are missing.
	MissingPVCount int32 `json:"missingPvCount"`

	// MetadataCount denotes number of metadata areas on the
	// volume group.
	MetadataCount int32 `json:"metadataCount"`

	// MetadataUsedCount denotes number of used metadata areas in
	// volume group
	MetadataUsedCount int32 `json:"metadataUsedCount"`

	// MetadataFree specifies the available metadata area space
	// for the volume group
	MetadataFree resource.Quantity `json:"metadataFree"`

	// MetadataSize specifies size of smallest metadata area
	// for the volume group
	MetadataSize resource.Quantity `json:"metadataSize"`

	// Permission indicates the volume group permission
	// which can be writable or read-only.
	// Permission has the following mapping between
	// int and string for its value:
	// [-1: "", 0: "writeable", 1: "read-only"]
	Permission int `json:"permissions"`

	// AllocationPolicy indicates the volume group allocation
	// policy.
	// AllocationPolicy has the following mapping between
	// int and string for its value:
	// [-1: "", 0: "normal", 1: "contiguous", 2: "cling", 3: "anywhere", 4: "inherited"]
	AllocationPolicy int `json:"allocationPolicy"`
}

// LogicalVolume specifies attributes of a given lv that exists on the node.
type LogicalVolume struct {
	// Name of the lvm logical volume(name: pvc-213ca1e6-e271-4ec8-875c-c7def3a4908d)
	Name string

	// Full name of the lvm logical volume (fullName: linuxlvmvg/pvc-213ca1e6-e271-4ec8-875c-c7def3a4908d)
	FullName string

	// UUID denotes a unique identity of a lvm logical volume.
	UUID string

	// Size specifies the total size of logical volume in Bytes
	Size int64

	// Path specifies LVM logical volume path
	Path string

	// DMPath specifies device mapper path
	DMPath string

	// LVM logical volume device
	Device string

	// Name of the VG in which LVM logical volume is created
	VGName string

	// SegType specifies the type of Logical volume segment
	SegType string

	// Permission indicates the logical volume permission.
	// Permission has the following mapping between
	// int and string for its value:
	// [-1: "", 0: "unknown", 1: "writeable", 2: "read-only", 3: "read-only-override"]
	Permission int

	// BehaviourWhenFull indicates the behaviour of thin pools when it is full.
	// BehaviourWhenFull has the following mapping between int and string for its value:
	// [-1: "", 0: "error", 1: "queue"]
	BehaviourWhenFull int

	// HealthStatus indicates the health status of logical volumes.
	// HealthStatus has the following mapping between int and string for its value:
	// [0: "", 1: "partial", 2: "refresh needed", 3: "mismatches exist"]
	HealthStatus int

	// RaidSyncAction indicates the current synchronization action being performed for RAID
	// action.
	// RaidSyncAction has the following mapping between int and string for its value:
	// [-1: "", 0: "idle", 1: "frozen", 2: "resync", 3: "recover", 4: "check", 5: "repair"]
	RaidSyncAction int

	// ActiveStatus indicates the active state of logical volume
	ActiveStatus string

	// Host specifies the creation host of the logical volume, if known
	Host string

	// For thin volumes, the thin pool Logical volume for that volume
	PoolName string

	// UsedSizePercent specifies the percentage full for snapshot, cache
	// and thin pools and volumes if logical volume is active.
	UsedSizePercent float64

	// MetadataSize specifies the size of the logical volume that holds
	// the metadata for thin and cache pools.
	MetadataSize int64

	// MetadataUsedPercent specifies the percentage of metadata full if logical volume
	// is active for cache and thin pools.
	MetadataUsedPercent float64

	// SnapshotUsedPercent specifies the percentage full for snapshots  if
	// logical volume is active.
	SnapshotUsedPercent float64
}

// PhysicalVolume specifies attributes of a given pv that exists on the node.
type PhysicalVolume struct {
	// Name of the lvm physical volume.
	Name string

	// UUID denotes a unique identity of a lvm physical volume.
	UUID string

	// Size specifies the total size of physical volume in bytes
	Size resource.Quantity

	// DeviceSize specifies the size of underlying device in bytes
	DeviceSize resource.Quantity

	// MetadataSize specifies the size of smallest metadata area on this device in bytes
	MetadataSize resource.Quantity

	// MetadataFree specifies the free metadata area space on the device in bytes
	MetadataFree resource.Quantity

	// Free specifies the physical volume unallocated space in bytes
	Free resource.Quantity

	// Used specifies the physical volume allocated space in bytes
	Used resource.Quantity

	// Allocatable indicates whether the device can be used for allocation
	Allocatable string

	// Missing indicates whether the device is missing in the system
	Missing string

	// InUse indicates whether or not the physical volume is in use
	InUse string

	// Name of the volume group which uses this physical volume
	VGName string
}

// VolumeInfo defines LVM info
type VolumeInfo struct {
	// OwnerNodeID is the Node ID where the volume group is present which is where
	// the volume has been provisioned.
	// OwnerNodeID can not be edited after the volume has been provisioned.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	OwnerNodeID string `json:"ownerNodeID"`

	// VolGroup specifies the name of the volume group where the volume has been created.
	// +kubebuilder:validation:Required
	VolGroup string `json:"volGroup"`

	// VgPattern specifies the regex to choose volume groups where volume
	// needs to be created.
	// +kubebuilder:validation:Required
	VgPattern string `json:"vgPattern"`

	// Capacity of the volume
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Capacity string `json:"capacity"`

	// Shared specifies whether the volume can be shared among multiple pods.
	// If it is not set to "yes", then the LVM LocalPV Driver will not allow
	// the volumes to be mounted by more than one pods.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=yes;no
	Shared string `json:"shared,omitempty"`

	// ThinProvision specifies whether logical volumes can be thinly provisioned.
	// If it is set to "yes", then the LVM LocalPV Driver will create
	// thinProvision i.e. logical volumes that are larger than the available extents.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=yes;no
	ThinProvision string `json:"thinProvision,omitempty"`
}
