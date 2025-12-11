package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/zhukovaskychina/xmongodb/config"
	"github.com/zhukovaskychina/xmongodb/server/protocol/bsoncore"
)

// Engine 存储引擎接口
// 这是对外的高层接口，内部使用 KVEngine 实现
type Engine interface {
	// 基础操作
	Start() error
	Stop() error
	Close() error

	// 数据库操作
	CreateDatabase(ctx context.Context, name string) error
	DropDatabase(ctx context.Context, name string) error
	ListDatabases(ctx context.Context) ([]string, error)

	// 集合操作
	CreateCollection(ctx context.Context, database, collection string) error
	DropCollection(ctx context.Context, database, collection string) error
	ListCollections(ctx context.Context, database string) ([]string, error)

	// 文档操作
	Insert(ctx context.Context, database, collection string, documents []Document) error
	Find(ctx context.Context, database, collection string, filter Document) ([]Document, error)
	Update(ctx context.Context, database, collection string, filter, update Document) error
	Delete(ctx context.Context, database, collection string, filter Document) error

	// 索引操作
	CreateIndex(ctx context.Context, database, collection string, index Index) error
	DropIndex(ctx context.Context, database, collection string, indexName string) error
	ListIndexes(ctx context.Context, database, collection string) ([]Index, error)

	// 统计信息
	GetStats() map[string]interface{}
}

// Document 文档类型
type Document map[string]interface{}

// Index 索引定义
type Index struct {
	Name   string
	Keys   map[string]int // 1: 升序, -1: 降序
	Unique bool
	Sparse bool
}

// NewEngine 创建新的存储引擎
func NewEngine(cfg config.StorageConfig) (Engine, error) {
	switch cfg.Engine {
	case "wiredTiger":
		return NewWiredTigerEngine(cfg)
	case "memory":
		return NewMemoryEngine(cfg)
	default:
		return nil, fmt.Errorf("不支持的存储引擎: %s", cfg.Engine)
	}
}

// WiredTigerEngine WiredTiger 存储引擎
// 基于 KVEngine 构建的高层存储引擎
type WiredTigerEngine struct {
	mu sync.RWMutex
	
	config    config.StorageConfig
	databases map[string]*Database
	running   bool
	
	// 底层 KV 引擎
	kvEngine KVEngine
	
	// 下一个 RecordId
	nextRecordId int64
}

// NewWiredTigerEngine 创建 WiredTiger 引擎
func NewWiredTigerEngine(cfg config.StorageConfig) (*WiredTigerEngine, error) {
	// 创建 KV 引擎配置
	kvConfig := KVEngineConfig{
		CacheSize:         1024 * 1024 * 1024, // 1GB
		MaxSessions:       1000,
		CheckpointEnabled: true,
	}
	
	return &WiredTigerEngine{
		config:    cfg,
		databases: make(map[string]*Database),
		kvEngine:  NewKVEngine(kvConfig),
	}, nil
}

// Start 启动引擎
func (e *WiredTigerEngine) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if e.running {
		return fmt.Errorf("存储引擎已经在运行")
	}

	// 启动底层 KV 引擎
	ctx := context.Background()
	if err := e.kvEngine.Start(ctx); err != nil {
		return fmt.Errorf("启动 KV 引擎失败: %w", err)
	}
	
	e.running = true
	return nil
}

// Stop 停止引擎
func (e *WiredTigerEngine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if !e.running {
		return nil
	}

	// 停止底层 KV 引擎
	ctx := context.Background()
	if err := e.kvEngine.Stop(ctx); err != nil {
		return fmt.Errorf("停止 KV 引擎失败: %w", err)
	}
	
	e.running = false
	return nil
}

// Close 关闭引擎
func (e *WiredTigerEngine) Close() error {
	return e.Stop()
}

// CreateDatabase 创建数据库
func (e *WiredTigerEngine) CreateDatabase(ctx context.Context, name string) error {
	if _, exists := e.databases[name]; exists {
		return fmt.Errorf("数据库 %s 已存在", name)
	}

	e.databases[name] = &Database{
		Name:        name,
		Collections: make(map[string]*Collection),
	}
	return nil
}

// DropDatabase 删除数据库
func (e *WiredTigerEngine) DropDatabase(ctx context.Context, name string) error {
	if _, exists := e.databases[name]; !exists {
		return fmt.Errorf("数据库 %s 不存在", name)
	}

	delete(e.databases, name)
	return nil
}

// ListDatabases 列出所有数据库
func (e *WiredTigerEngine) ListDatabases(ctx context.Context) ([]string, error) {
	databases := make([]string, 0, len(e.databases))
	for name := range e.databases {
		databases = append(databases, name)
	}
	return databases, nil
}

// CreateCollection 创建集合
func (e *WiredTigerEngine) CreateCollection(ctx context.Context, database, collection string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	db, exists := e.databases[database]
	if !exists {
		return fmt.Errorf("数据库 %s 不存在", database)
	}

	if _, exists := db.Collections[collection]; exists {
		return fmt.Errorf("集合 %s 已存在", collection)
	}
	
	// 创建 RecordStore
	namespace := makeNamespace(database, collection)
	recordStore, err := e.kvEngine.CreateRecordStore(namespace)
	if err != nil {
		return fmt.Errorf("创建 RecordStore 失败: %w", err)
	}
	
	// 创建默认的 _id 索引
	idxName := "_id_"
	idIndex, err := e.kvEngine.CreateSortedDataInterface(namespace, idxName, true)
	if err != nil {
		return fmt.Errorf("创建 _id 索引失败: %w", err)
	}

	db.Collections[collection] = &Collection{
		Name:        collection,
		RecordStore: recordStore,
		Indexes:     make(map[string]SortedDataInterface),
	}
	db.Collections[collection].Indexes[idxName] = idIndex
	
	return nil
}

// DropCollection 删除集合
func (e *WiredTigerEngine) DropCollection(ctx context.Context, database, collection string) error {
	db, exists := e.databases[database]
	if !exists {
		return fmt.Errorf("数据库 %s 不存在", database)
	}

	if _, exists := db.Collections[collection]; !exists {
		return fmt.Errorf("集合 %s 不存在", collection)
	}

	delete(db.Collections, collection)
	return nil
}

// ListCollections 列出集合
func (e *WiredTigerEngine) ListCollections(ctx context.Context, database string) ([]string, error) {
	db, exists := e.databases[database]
	if !exists {
		return nil, fmt.Errorf("数据库 %s 不存在", database)
	}

	collections := make([]string, 0, len(db.Collections))
	for name := range db.Collections {
		collections = append(collections, name)
	}
	return collections, nil
}

// Insert 插入文档
func (e *WiredTigerEngine) Insert(ctx context.Context, database, collection string, documents []Document) error {
	e.mu.RLock()
	db, exists := e.databases[database]
	if !exists {
		e.mu.RUnlock()
		return fmt.Errorf("数据库 %s 不存在", database)
	}

	coll, exists := db.Collections[collection]
	if !exists {
		e.mu.RUnlock()
		return fmt.Errorf("集合 %s 不存在", collection)
	}
	e.mu.RUnlock()
	
	// 插入每个文档
	for _, doc := range documents {
		// 生成 RecordId
		recordId := NewRecordIdFromLong(atomic.AddInt64(&e.nextRecordId, 1))
		
		// 确保文档有 _id 字段
		if _, hasId := doc["_id"]; !hasId {
			doc["_id"] = recordId.String()
		}
		
		// 将文档序列化为 BSON
		data, err := e.documentToBSON(doc)
		if err != nil {
			return fmt.Errorf("序列化文档失败: %w", err)
		}
		
		// 插入到 RecordStore
		if err := coll.RecordStore.InsertRecord(ctx, recordId, data); err != nil {
			return fmt.Errorf("插入记录失败: %w", err)
		}
		
		// 更新索引
		for _, idx := range coll.Indexes {
			// 提取索引键（简化实现，这里使用 _id）
			idxKey := []byte(doc["_id"].(string))
			if err := idx.Insert(ctx, idxKey, recordId); err != nil {
				return fmt.Errorf("更新索引失败: %w", err)
			}
		}
	}
	
	return nil
}

// Find 查找文档
func (e *WiredTigerEngine) Find(ctx context.Context, database, collection string, filter Document) ([]Document, error) {
	e.mu.RLock()
	db, exists := e.databases[database]
	if !exists {
		e.mu.RUnlock()
		return nil, fmt.Errorf("数据库 %s 不存在", database)
	}

	coll, exists := db.Collections[collection]
	if !exists {
		e.mu.RUnlock()
		return nil, fmt.Errorf("集合 %s 不存在", collection)
	}
	e.mu.RUnlock()

	// 扫描所有记录（简化实现）
	cursor, err := coll.RecordStore.Scan(ctx, NullRecordId())
	if err != nil {
		return nil, fmt.Errorf("扫描记录失败: %w", err)
	}
	defer cursor.Close()
	
	results := make([]Document, 0)
	for cursor.Next() {
		data := cursor.Data()
		
		// 将 BSON 反序列化为文档
		doc, err := e.bsonToDocument(data)
		if err != nil {
			continue
		}
		
		// TODO: 应用过滤器
		results = append(results, doc)
	}
	
	return results, nil
}

// Update 更新文档
func (e *WiredTigerEngine) Update(ctx context.Context, database, collection string, filter, update Document) error {
	// TODO: 实现更新逻辑
	return nil
}

// Delete 删除文档
func (e *WiredTigerEngine) Delete(ctx context.Context, database, collection string, filter Document) error {
	// TODO: 实现删除逻辑
	return nil
}

// CreateIndex 创建索引
func (e *WiredTigerEngine) CreateIndex(ctx context.Context, database, collection string, index Index) error {
	// TODO: 实现索引创建逻辑
	return nil
}

// DropIndex 删除索引
func (e *WiredTigerEngine) DropIndex(ctx context.Context, database, collection string, indexName string) error {
	// TODO: 实现索引删除逻辑
	return nil
}

// ListIndexes 列出索引
func (e *WiredTigerEngine) ListIndexes(ctx context.Context, database, collection string) ([]Index, error) {
	// TODO: 实现索引列出逻辑
	return nil, nil
}

// GetStats 获取统计信息
func (e *WiredTigerEngine) GetStats() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	stats := make(map[string]interface{})
	stats["engine"] = "wiredTiger"
	stats["running"] = e.running
	stats["databases"] = len(e.databases)
	
	// 添加 KV 引擎统计
	if e.kvEngine != nil {
		stats["kv_engine"] = e.kvEngine.GetStats()
	}
	
	return stats
}

// Database 数据库结构
type Database struct {
	Name        string
	Collections map[string]*Collection
}

// Collection 集合结构
type Collection struct {
	Name        string
	RecordStore RecordStore                     // B+Tree 记录存储
	Indexes     map[string]SortedDataInterface // 索引映射
}

// MemoryEngine 内存存储引擎
type MemoryEngine struct {
	*WiredTigerEngine
}

// NewMemoryEngine 创建内存引擎
func NewMemoryEngine(cfg config.StorageConfig) (*MemoryEngine, error) {
	wt, err := NewWiredTigerEngine(cfg)
	if err != nil {
		return nil, err
	}

	return &MemoryEngine{
		WiredTigerEngine: wt,
	}, nil
}

// makeNamespace 创建命名空间
func makeNamespace(database, collection string) string {
	return database + "." + collection
}

// documentToBSON 将 Document 转换为 BSON 字节数组
func (e *WiredTigerEngine) documentToBSON(doc Document) ([]byte, error) {
	// 简化实现：使用 JSON 编码
	// TODO: 在生产环境中应该使用真正的 BSON 编码
	return json.Marshal(doc)
}

// bsonToDocument 将 BSON 字节数组转换为 Document
func (e *WiredTigerEngine) bsonToDocument(data []byte) (Document, error) {
	// 简化实现：使用 JSON 解码
	// TODO: 在生产环境中应该使用真正的 BSON 解码
	var doc Document
	err := json.Unmarshal(data, &doc)
	return doc, err
}
