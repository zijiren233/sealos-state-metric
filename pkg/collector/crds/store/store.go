// pkg/collector/crds/store/resource_store.go
package store

import (
	"fmt"
	"sync"
	"sync/atomic"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ResourceStore 资源存储层
// 职责：管理所有 CRD 资源的内存缓存，提供线程安全的 CRUD 操作
type ResourceStore struct {
	// 资源存储：key 格式为 "namespace/name" 或 "name"（cluster-scoped）
	mu        sync.RWMutex
	resources map[string]*unstructured.Unstructured

	// 日志
	logger *log.Entry

	// 配置
	config *StoreConfig

	// 统计信息
	metrics StoreMetrics
}

// StoreConfig 存储配置
type StoreConfig struct {
	// 是否启用深拷贝（防止外部修改）
	// 建议启用，虽然有性能开销，但保证数据安全
	EnableDeepCopy bool

	// 资源类型标识（用于日志和调试）
	// 例如："metering.sealos.io/v1/Account"
	ResourceType string

	// 最大缓存数量（0 表示无限制）
	// 超过此数量时会记录警告日志
	MaxSize int
}

// StoreMetrics 存储层统计信息
type StoreMetrics struct {
	AddCount    atomic.Int64 // 添加操作计数
	UpdateCount atomic.Int64 // 更新操作计数
	DeleteCount atomic.Int64 // 删除操作计数
	TotalCount  atomic.Int64 // 当前总数
}

// NewResourceStore 创建新的资源存储
func NewResourceStore(logger *log.Entry, config *StoreConfig) *ResourceStore {
	if config == nil {
		config = &StoreConfig{
			EnableDeepCopy: true, // 默认启用深拷贝
			ResourceType:   "unknown",
			MaxSize:        0, // 无限制
		}
	}

	if logger == nil {
		logger = log.WithField("component", "resource_store")
	}

	store := &ResourceStore{
		resources: make(map[string]*unstructured.Unstructured),
		logger:    logger.WithField("resource_type", config.ResourceType),
		config:    config,
	}

	store.logger.WithFields(log.Fields{
		"deep_copy": config.EnableDeepCopy,
		"max_size":  config.MaxSize,
	}).Info("Resource store initialized")

	return store
}

// Add 添加资源到存储
// 如果资源已存在，会被覆盖（等同于 Update）
func (s *ResourceStore) Add(obj *unstructured.Unstructured) {
	if obj == nil {
		s.logger.Warn("Attempted to add nil object")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.getKey(obj)

	// 检查是否已存在
	_, exists := s.resources[key]

	// 存储对象（可选深拷贝）
	if s.config.EnableDeepCopy {
		s.resources[key] = obj.DeepCopy()
	} else {
		s.resources[key] = obj
	}

	// 更新统计
	if exists {
		s.metrics.UpdateCount.Add(1)
		s.logger.WithField("key", key).Debug("Resource updated (via Add)")
	} else {
		s.metrics.AddCount.Add(1)
		s.metrics.TotalCount.Add(1)
		s.logger.WithField("key", key).Debug("Resource added")
	}

	// 检查大小限制
	s.checkSizeLimit()
}

// Update 更新存储中的资源
// 如果资源不存在，会被添加（等同于 Add）
func (s *ResourceStore) Update(obj *unstructured.Unstructured) {
	if obj == nil {
		s.logger.Warn("Attempted to update nil object")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.getKey(obj)

	// 检查是否已存在
	_, exists := s.resources[key]

	// 存储对象（可选深拷贝）
	if s.config.EnableDeepCopy {
		s.resources[key] = obj.DeepCopy()
	} else {
		s.resources[key] = obj
	}

	// 更新统计
	if exists {
		s.metrics.UpdateCount.Add(1)
		s.logger.WithField("key", key).Debug("Resource updated")
	} else {
		s.metrics.AddCount.Add(1)
		s.metrics.TotalCount.Add(1)
		s.logger.WithField("key", key).Debug("Resource added (via Update)")
	}

	// 检查大小限制
	s.checkSizeLimit()
}

// Delete 从存储中删除资源
func (s *ResourceStore) Delete(obj *unstructured.Unstructured) {
	if obj == nil {
		s.logger.Warn("Attempted to delete nil object")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.getKey(obj)

	// 检查是否存在
	if _, exists := s.resources[key]; exists {
		delete(s.resources, key)
		s.metrics.DeleteCount.Add(1)
		s.metrics.TotalCount.Add(-1)
		s.logger.WithField("key", key).Debug("Resource deleted")
	} else {
		s.logger.WithField("key", key).Debug("Resource not found for deletion")
	}
}

// Get 根据 namespace 和 name 获取资源
// 返回资源对象和是否存在的标志
func (s *ResourceStore) Get(namespace, name string) (*unstructured.Unstructured, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := s.makeKey(namespace, name)
	obj, exists := s.resources[key]

	if !exists {
		return nil, false
	}

	// 返回深拷贝，防止外部修改
	if s.config.EnableDeepCopy {
		return obj.DeepCopy(), true
	}

	return obj, true
}

// List 返回所有资源的列表
// 返回的是资源的副本（如果启用了深拷贝）
func (s *ResourceStore) List() []*unstructured.Unstructured {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*unstructured.Unstructured, 0, len(s.resources))

	for _, obj := range s.resources {
		if s.config.EnableDeepCopy {
			result = append(result, obj.DeepCopy())
		} else {
			result = append(result, obj)
		}
	}

	return result
}

// ListByNamespace 返回指定 namespace 下的所有资源
// 如果 namespace 为空字符串，返回所有 cluster-scoped 资源
func (s *ResourceStore) ListByNamespace(namespace string) []*unstructured.Unstructured {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*unstructured.Unstructured, 0)

	for _, obj := range s.resources {
		if obj.GetNamespace() == namespace {
			if s.config.EnableDeepCopy {
				result = append(result, obj.DeepCopy())
			} else {
				result = append(result, obj)
			}
		}
	}

	return result
}

// Len 返回当前存储的资源数量
func (s *ResourceStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.resources)
}

// Clear 清空所有资源
// 注意：此操作不可逆，谨慎使用
func (s *ResourceStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := len(s.resources)
	s.resources = make(map[string]*unstructured.Unstructured)
	s.metrics.TotalCount.Store(0)

	s.logger.WithField("cleared_count", count).Info("Resource store cleared")
}

// GetMetrics 获取存储统计信息的快照
func (s *ResourceStore) GetMetrics() StoreMetricsSnapshot {
	return StoreMetricsSnapshot{
		AddCount:    s.metrics.AddCount.Load(),
		UpdateCount: s.metrics.UpdateCount.Load(),
		DeleteCount: s.metrics.DeleteCount.Load(),
		TotalCount:  s.metrics.TotalCount.Load(),
	}
}

// StoreMetricsSnapshot 统计信息快照（用于返回）
type StoreMetricsSnapshot struct {
	AddCount    int64
	UpdateCount int64
	DeleteCount int64
	TotalCount  int64
}

// GetResourceType 获取资源类型标识
func (s *ResourceStore) GetResourceType() string {
	return s.config.ResourceType
}

// IsEmpty 检查存储是否为空
func (s *ResourceStore) IsEmpty() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.resources) == 0
}

// Contains 检查指定资源是否存在
func (s *ResourceStore) Contains(namespace, name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := s.makeKey(namespace, name)
	_, exists := s.resources[key]
	return exists
}

// getKey 从 Unstructured 对象生成存储键
// 内部方法，调用前必须已持有锁
func (s *ResourceStore) getKey(obj *unstructured.Unstructured) string {
	namespace := obj.GetNamespace()
	name := obj.GetName()
	return s.makeKey(namespace, name)
}

// makeKey 根据 namespace 和 name 生成存储键
// 格式：
//   - namespace-scoped: "namespace/name"
//   - cluster-scoped: "name"
func (s *ResourceStore) makeKey(namespace, name string) string {
	if namespace == "" {
		// Cluster-scoped 资源
		return name
	}
	// Namespace-scoped 资源
	return fmt.Sprintf("%s/%s", namespace, name)
}

// checkSizeLimit 检查存储大小是否超过限制
// 内部方法，调用前必须已持有锁
func (s *ResourceStore) checkSizeLimit() {
	if s.config.MaxSize > 0 && len(s.resources) > s.config.MaxSize {
		s.logger.WithFields(log.Fields{
			"current_size": len(s.resources),
			"max_size":     s.config.MaxSize,
		}).Warn("Resource store size exceeded limit")
	}
}

// LogStats 输出当前存储统计信息（用于调试）
func (s *ResourceStore) LogStats() {
	metrics := s.GetMetrics()
	s.logger.WithFields(log.Fields{
		"total_count":  metrics.TotalCount,
		"add_count":    metrics.AddCount,
		"update_count": metrics.UpdateCount,
		"delete_count": metrics.DeleteCount,
	}).Info("Resource store statistics")
}

// GetKeys 返回所有存储的键（用于调试）
func (s *ResourceStore) GetKeys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.resources))
	for key := range s.resources {
		keys = append(keys, key)
	}
	return keys
}
