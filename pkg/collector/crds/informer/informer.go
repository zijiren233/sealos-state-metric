// pkg/collector/crds/informer.go
package informer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/labring/sealos-state-metrics/pkg/collector/crds/store"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

// Informer K8s 资源监听器
// 职责：监听 K8s API 资源变化并同步到 ResourceStore
type Informer struct {
	// K8s 相关
	config        *InformerConfig
	dynamicClient dynamic.Interface
	informer      cache.SharedIndexInformer

	// 生命周期管理
	informerStopCh chan struct{}
	started        bool

	// 存储层引用（核心依赖）
	store *store.ResourceStore

	// 日志
	logger *log.Entry
}

// InformerConfig 监听器配置
type InformerConfig struct {
	// GVR 是要监听的资源类型
	GVR schema.GroupVersionResource

	// 重新同步周期
	ResyncPeriod time.Duration
}

// NewInformer 创建新的 Informer
// 参数：
//   - dynamicClient: K8s 动态客户端
//   - config: Informer 配置
//   - resourceStore: 资源存储层（依赖注入）
//   - logger: 日志记录器
func NewInformer(
	dynamicClient dynamic.Interface,
	config *InformerConfig,
	resourceStore *store.ResourceStore,
	logger *log.Entry,
) (*Informer, error) {
	// 参数验证
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}

	if dynamicClient == nil {
		return nil, errors.New("dynamic client cannot be nil")
	}

	if resourceStore == nil {
		return nil, errors.New("resource store cannot be nil")
	}
	if config.ResyncPeriod == 0 {
		config.ResyncPeriod = 10 * time.Minute
	}
	if logger == nil {
		logger = log.WithField("component", "informer")
	}
	informerLogger := logger.WithFields(log.Fields{
		"gvr": config.GVR.String(),
	})

	return &Informer{
		config:        config,
		dynamicClient: dynamicClient,
		store:         resourceStore,
		logger:        informerLogger,
		started:       false,
	}, nil
}

// Start 启动 Informer 并开始监听资源
func (i *Informer) Start(ctx context.Context) error {
	if i.started {
		return errors.New("informer already started")
	}

	i.logger.Info("Starting dynamic informer")
	var factory dynamicinformer.DynamicSharedInformerFactory
	factory = dynamicinformer.NewDynamicSharedInformerFactory(
		i.dynamicClient,
		i.config.ResyncPeriod,
	)
	i.logger.Debug("Created cluster-scoped informer factory")
	i.informer = factory.ForResource(i.config.GVR).Informer()
	if err := i.registerEventHandlers(); err != nil {
		return fmt.Errorf("failed to register event handlers: %w", err)
	}

	i.informerStopCh = make(chan struct{})
	go i.informer.Run(i.informerStopCh)

	// 等待缓存同步
	i.logger.Info("Waiting for informer cache to sync")
	if !cache.WaitForCacheSync(ctx.Done(), i.informer.HasSynced) {
		close(i.informerStopCh)
		i.informerStopCh = nil
		return errors.New("failed to sync informer cache")
	}

	i.started = true
	i.logger.Info("Dynamic informer started and cache synced successfully")

	// 输出初始统计信息
	i.logInitialStats()

	return nil
}

// Stop 停止 Informer
func (i *Informer) Stop() error {
	if !i.started {
		i.logger.Warn("Informer not started, nothing to stop")
		return nil
	}

	i.logger.Info("Stopping dynamic informer")

	if i.informerStopCh != nil {
		close(i.informerStopCh)
		i.informerStopCh = nil
	}

	i.started = false
	i.logger.Info("Dynamic informer stopped")

	return nil
}

// HasSynced 返回 Informer 缓存是否已同步
func (i *Informer) HasSynced() bool {
	if i.informer == nil {
		return false
	}
	return i.informer.HasSynced()
}

// IsStarted 返回 Informer 是否已启动
func (i *Informer) IsStarted() bool {
	return i.started
}

// GetStore 返回 Informer 内部的 cache.Store（用于调试）
// 注意：不建议直接使用，应该通过 ResourceStore 访问数据
func (i *Informer) GetStore() cache.Store {
	if i.informer == nil {
		return nil
	}
	return i.informer.GetStore()
}

// GetConfig 返回 Informer 配置
func (i *Informer) GetConfig() *InformerConfig {
	return i.config
}

// registerEventHandlers 注册事件处理器
func (i *Informer) registerEventHandlers() error {
	_, err := i.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			u := i.extractUnstructured(obj)
			if u != nil {
				i.handleAdd(u)
			}
		},
		UpdateFunc: func(oldObj, newObj any) {
			oldU := i.extractUnstructured(oldObj)
			newU := i.extractUnstructured(newObj)
			if oldU != nil && newU != nil {
				i.handleUpdate(oldU, newU)
			}
		},
		DeleteFunc: func(obj any) {
			u := i.extractUnstructured(obj)
			if u != nil {
				i.handleDelete(u)
			}
		},
	})

	if err != nil {
		return fmt.Errorf("failed to add event handler: %w", err)
	}

	i.logger.Debug("Event handlers registered successfully")
	return nil
}

// handleAdd 处理资源添加事件
func (i *Informer) handleAdd(obj *unstructured.Unstructured) {
	i.logger.WithFields(log.Fields{
		"namespace": obj.GetNamespace(),
		"name":      obj.GetName(),
		"uid":       obj.GetUID(),
	}).Debug("Resource added")

	// 直接写入 ResourceStore
	i.store.Add(obj)
}

// handleUpdate 处理资源更新事件
func (i *Informer) handleUpdate(oldObj, newObj *unstructured.Unstructured) {
	// 检查 ResourceVersion 是否变化（避免无效更新）
	if oldObj.GetResourceVersion() == newObj.GetResourceVersion() {
		i.logger.WithFields(log.Fields{
			"namespace": newObj.GetNamespace(),
			"name":      newObj.GetName(),
		}).Debug("Resource update ignored (same resource version)")
		return
	}

	i.logger.WithFields(log.Fields{
		"namespace":   newObj.GetNamespace(),
		"name":        newObj.GetName(),
		"uid":         newObj.GetUID(),
		"old_version": oldObj.GetResourceVersion(),
		"new_version": newObj.GetResourceVersion(),
	}).Debug("Resource updated")

	// 直接写入 ResourceStore
	i.store.Update(newObj)
}

// handleDelete 处理资源删除事件
func (i *Informer) handleDelete(obj *unstructured.Unstructured) {
	i.logger.WithFields(log.Fields{
		"namespace": obj.GetNamespace(),
		"name":      obj.GetName(),
		"uid":       obj.GetUID(),
	}).Debug("Resource deleted")

	// 从 ResourceStore 删除
	i.store.Delete(obj)
}

// extractUnstructured 从事件对象中提取 Unstructured
// 处理 DeletedFinalStateUnknown 的特殊情况
func (i *Informer) extractUnstructured(obj any) *unstructured.Unstructured {
	// 直接类型断言
	if u, ok := obj.(*unstructured.Unstructured); ok {
		return u
	}

	// 处理 DeletedFinalStateUnknown（删除事件的特殊情况）
	if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		if u, ok := tombstone.Obj.(*unstructured.Unstructured); ok {
			i.logger.WithFields(log.Fields{
				"namespace": u.GetNamespace(),
				"name":      u.GetName(),
			}).Debug("Extracted object from tombstone")
			return u
		}

		i.logger.WithField("object", tombstone.Obj).
			Error("Tombstone contained object that is not Unstructured")
		return nil
	}

	// 无法识别的类型
	i.logger.WithField("object_type", fmt.Sprintf("%T", obj)).
		Error("Failed to extract Unstructured from object")
	return nil
}

// logInitialStats 输出初始统计信息
func (i *Informer) logInitialStats() {
	storeLen := i.store.Len()
	cacheLen := 0
	if i.informer != nil && i.informer.GetStore() != nil {
		cacheLen = len(i.informer.GetStore().List())
	}

	i.logger.WithFields(log.Fields{
		"store_count": storeLen,
		"cache_count": cacheLen,
	}).Info("Initial sync completed")
}

// GetResourceCount 获取当前监听的资源数量
func (i *Informer) GetResourceCount() int {
	return i.store.Len()
}

// LogStats 输出当前统计信息（用于调试和监控）
func (i *Informer) LogStats() {
	if !i.started {
		i.logger.Warn("Informer not started")
		return
	}

	metrics := i.store.GetMetrics()

	i.logger.WithFields(log.Fields{
		"synced":       i.HasSynced(),
		"store_count":  metrics.TotalCount,
		"add_count":    metrics.AddCount,
		"update_count": metrics.UpdateCount,
		"delete_count": metrics.DeleteCount,
	}).Info("Informer statistics")
}

// WaitForSync 等待 Informer 缓存同步（带超时）
func (i *Informer) WaitForSync(timeout time.Duration) error {
	if !i.started {
		return errors.New("informer not started")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if !cache.WaitForCacheSync(ctx.Done(), i.informer.HasSynced) {
		return fmt.Errorf("failed to sync cache within %v", timeout)
	}

	return nil
}

func (i *Informer) Resync() error {
	if !i.started {
		return errors.New("informer not started")
	}
	if i.informer == nil {
		return errors.New("informer not initialized")
	}
	i.logger.Info("Triggering manual resync")
	return nil
}
