package server

import "github.com/prometheus/client_golang/prometheus"

// ReloadAwareCollector wraps a prometheus.Collector and blocks operations during reload
type ReloadAwareCollector struct {
	server *Server
	inner  prometheus.Collector
}

// Describe implements prometheus.Collector
func (rc *ReloadAwareCollector) Describe(ch chan<- *prometheus.Desc) {
	// Hold server read lock - allows concurrent describes but blocks during reload
	rc.server.mu.RLock()
	defer rc.server.mu.RUnlock()

	// Delegate to inner collector
	rc.inner.Describe(ch)
}

// Collect implements prometheus.Collector
func (rc *ReloadAwareCollector) Collect(ch chan<- prometheus.Metric) {
	// Hold server read lock - allows concurrent collection but blocks during reload
	rc.server.mu.RLock()
	defer rc.server.mu.RUnlock()

	// Delegate to inner collector
	rc.inner.Collect(ch)
}
