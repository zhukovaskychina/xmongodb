package storage

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	
	"github.com/google/uuid"
)

// KVEngine 键值存储引擎接口
// 类似于 WiredTigerKVEngine，作为底层存储引擎
type KVEngine interface {
	// 生命周期管理
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	
	// 会话管理
	CreateSession(ctx context.Context) (EngineSession, error)
	
	// RecordStore 管理
	GetRecordStore(namespace string) (RecordStore, error)
	CreateRecordStore(namespace string) (RecordStore, error)
	DropRecordStore(namespace string) error
	
	// SortedDataInterface（索引）管理
	GetSortedDataInterface(namespace, indexName string) (SortedDataInterface, error)
	CreateSortedDataInterface(namespace, indexName string, unique bool) (SortedDataInterface, error)
	DropSortedDataInterface(namespace, indexName string) error
	
	// 统计信息
	GetStats() map[string]interface{}
}

// WiredTigerKVEngine WiredTiger 风格的 KV 引擎实现
type WiredTigerKVEngine struct {
	mu sync.RWMutex
	
	// 运行状态
	running bool
	
	// RecordStore 管理
	// namespace -> RecordStore
	recordStores map[string]RecordStore
	
	// SortedDataInterface（索引）管理
	// namespace.indexName -> SortedDataInterface
	indexes map[string]SortedDataInterface
	
	// 会话管理
	sessions     map[string]EngineSession
	sessionCount int64
	
	// 配置
	config KVEngineConfig
}

// KVEngineConfig KV 引擎配置
type KVEngineConfig struct {
	// 缓存大小（字节）
	CacheSize int64
	
	// 最大会话数
	MaxSessions int
	
	// 是否启用检查点
	CheckpointEnabled bool
}

// NewKVEngine 创建新的 KV 引擎
func NewKVEngine(config KVEngineConfig) KVEngine {
	if config.MaxSessions == 0 {
		config.MaxSessions = 1000
	}
	if config.CacheSize == 0 {
		config.CacheSize = 1024 * 1024 * 1024 // 默认 1GB
	}
	
	return &WiredTigerKVEngine{
		recordStores: make(map[string]RecordStore),
		indexes:      make(map[string]SortedDataInterface),
		sessions:     make(map[string]EngineSession),
		config:       config,
	}
}

// Start 启动引擎
func (e *WiredTigerKVEngine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if e.running {
		return fmt.Errorf("KV 引擎已经在运行")
	}
	
	e.running = true
	return nil
}

// Stop 停止引擎
func (e *WiredTigerKVEngine) Stop(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if !e.running {
		return nil
	}
	
	// 关闭所有会话
	for _, session := range e.sessions {
		session.End(ctx)
	}
	e.sessions = make(map[string]EngineSession)
	
	e.running = false
	return nil
}

// CreateSession 创建会话
func (e *WiredTigerKVEngine) CreateSession(ctx context.Context) (EngineSession, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if !e.running {
		return nil, fmt.Errorf("KV 引擎未运行")
	}
	
	// 检查会话数限制
	if len(e.sessions) >= e.config.MaxSessions {
		return nil, fmt.Errorf("超过最大会话数限制: %d", e.config.MaxSessions)
	}
	
	// 生成唯一会话 ID
	sessionId := uuid.New().String()
	
	// 创建会话
	session := NewEngineSession(sessionId, e)
	if err := session.Begin(ctx); err != nil {
		return nil, fmt.Errorf("启动会话失败: %w", err)
	}
	
	e.sessions[sessionId] = session
	atomic.AddInt64(&e.sessionCount, 1)
	
	return session, nil
}

// GetRecordStore 获取 RecordStore
func (e *WiredTigerKVEngine) GetRecordStore(namespace string) (RecordStore, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	rs, exists := e.recordStores[namespace]
	if !exists {
		return nil, fmt.Errorf("RecordStore %s 不存在", namespace)
	}
	
	return rs, nil
}

// CreateRecordStore 创建 RecordStore
func (e *WiredTigerKVEngine) CreateRecordStore(namespace string) (RecordStore, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if _, exists := e.recordStores[namespace]; exists {
		return nil, fmt.Errorf("RecordStore %s 已存在", namespace)
	}
	
	rs := NewRecordStore(namespace)
	e.recordStores[namespace] = rs
	
	return rs, nil
}

// DropRecordStore 删除 RecordStore
func (e *WiredTigerKVEngine) DropRecordStore(namespace string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if _, exists := e.recordStores[namespace]; !exists {
		return fmt.Errorf("RecordStore %s 不存在", namespace)
	}
	
	delete(e.recordStores, namespace)
	
	// 同时删除相关的索引
	for key := range e.indexes {
		if len(key) > len(namespace) && key[:len(namespace)] == namespace {
			delete(e.indexes, key)
		}
	}
	
	return nil
}

// GetSortedDataInterface 获取索引
func (e *WiredTigerKVEngine) GetSortedDataInterface(namespace, indexName string) (SortedDataInterface, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	key := makeIndexKey(namespace, indexName)
	idx, exists := e.indexes[key]
	if !exists {
		return nil, fmt.Errorf("索引 %s.%s 不存在", namespace, indexName)
	}
	
	return idx, nil
}

// CreateSortedDataInterface 创建索引
func (e *WiredTigerKVEngine) CreateSortedDataInterface(namespace, indexName string, unique bool) (SortedDataInterface, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	key := makeIndexKey(namespace, indexName)
	if _, exists := e.indexes[key]; exists {
		return nil, fmt.Errorf("索引 %s.%s 已存在", namespace, indexName)
	}
	
	idx := NewSortedDataInterface(indexName, unique)
	e.indexes[key] = idx
	
	return idx, nil
}

// DropSortedDataInterface 删除索引
func (e *WiredTigerKVEngine) DropSortedDataInterface(namespace, indexName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	key := makeIndexKey(namespace, indexName)
	if _, exists := e.indexes[key]; !exists {
		return fmt.Errorf("索引 %s.%s 不存在", namespace, indexName)
	}
	
	delete(e.indexes, key)
	return nil
}

// GetStats 获取统计信息
func (e *WiredTigerKVEngine) GetStats() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	stats := make(map[string]interface{})
	stats["running"] = e.running
	stats["record_stores"] = len(e.recordStores)
	stats["indexes"] = len(e.indexes)
	stats["sessions"] = len(e.sessions)
	stats["total_sessions_created"] = atomic.LoadInt64(&e.sessionCount)
	stats["cache_size"] = e.config.CacheSize
	stats["max_sessions"] = e.config.MaxSessions
	
	// RecordStore 统计
	var totalRecords, totalDataSize int64
	for _, rs := range e.recordStores {
		totalRecords += rs.NumRecords()
		totalDataSize += rs.DataSize()
	}
	stats["total_records"] = totalRecords
	stats["total_data_size"] = totalDataSize
	
	// 索引统计
	var totalIndexEntries int64
	for _, idx := range e.indexes {
		totalIndexEntries += idx.NumEntries()
	}
	stats["total_index_entries"] = totalIndexEntries
	
	return stats
}

// makeIndexKey 创建索引键
func makeIndexKey(namespace, indexName string) string {
	return namespace + "." + indexName
}
