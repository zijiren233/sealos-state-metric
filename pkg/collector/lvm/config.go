package lvm

import (
	"time"
)

// Config contains configuration for the LVM collector
type Config struct {
	// UpdateInterval is the interval between LVM metrics updates
	UpdateInterval time.Duration `yaml:"updateInterval" env:"UPDATE_INTERVAL"`

	// NodeName is the name of the node to monitor (usually set from downward API)
	NodeName string `yaml:"nodeName" env:"NODE_NAME"`
}

// NewDefaultConfig returns the default configuration for LVM collector
// This function only returns hard-coded defaults without any env parsing
func NewDefaultConfig() *Config {
	return &Config{
		UpdateInterval: 10 * time.Second,
		NodeName:       "",
	}
}
