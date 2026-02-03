package crds

import (
	"sync"

	"github.com/labring/sealos-state-metrics/pkg/collector/base"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// Collector collects crd (Custom Resource Definition) metrics
type Collector struct {
	*base.BaseCollector
	crdCollectors  []*CrdCollector
	client         kubernetes.Interface
	stopCh         chan struct{}
	logger         *log.Entry
	mu             sync.RWMutex
	nodes          map[string]*corev1.Node // key: node name
	nodeHasMetrics map[string]bool         // key: node name, value: has metrics
}

func (c *Collector) collect(ch chan<- prometheus.Metric) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, crdCollector := range c.crdCollectors {
		crdCollector.Collect(ch)
	}
}
