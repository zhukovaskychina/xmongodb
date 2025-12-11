package storage

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	
	"github.com/zhukovaskychina/xmongodb/server/storage/btree"
)

// RecordStore 记录存储接口
// 负责存储完整的 BSON 文档，使用 RecordId 作为主键
type RecordStore interface {
	// 文档操作
	InsertRecord(ctx context.Context, recordId RecordId, data []byte) error
	UpdateRecord(ctx context.Context, recordId RecordId, data []byte) error
	DeleteRecord(ctx context.Context, recordId RecordId) error
	GetRecord(ctx context.Context, recordId RecordId) ([]byte, error)
	
	// 扫描操作
	Scan(ctx context.Context, startId RecordId) (RecordCursor, error)
	
	// 统计信息
	NumRecords() int64
	DataSize() int64
	
	// 生命周期
	Truncate(ctx context.Context) error
}

// RecordCursor 记录游标
type RecordCursor interface {
	Next() bool
	RecordId() RecordId
	Data() []byte
	Close() error
}

// BTreeRecordStore 基于 B+Tree 的记录存储实现
type BTreeRecordStore struct {
	mu sync.RWMutex
	
	// B+Tree 存储
	tree *btree.BTree
	
	// 统计信息
	numRecords int64
	dataSize   int64
	
	// 标识
	namespace string // database.collection
}

// NewRecordStore 创建新的 RecordStore
func NewRecordStore(namespace string) RecordStore {
	return &BTreeRecordStore{
		tree:      btree.NewBTree(128), // 使用阶数128的B+树
		namespace: namespace,
	}
}

// InsertRecord 插入记录
func (rs *BTreeRecordStore) InsertRecord(ctx context.Context, recordId RecordId, data []byte) error {
	if recordId.IsNull() {
		return fmt.Errorf("RecordId 不能为空")
	}
	
	// 将 RecordId 转换为字节数组作为键
	key, ok := recordId.AsBytes()
	if !ok {
		return fmt.Errorf("无法将 RecordId 转换为字节")
	}
	
	// 检查是否已存在
	if _, exists := rs.tree.Get(key); exists {
		return fmt.Errorf("RecordId %s 已存在", recordId.String())
	}
	
	// 插入到 B+Tree
	if err := rs.tree.Insert(key, data); err != nil {
		return fmt.Errorf("插入记录失败: %w", err)
	}
	
	// 更新统计
	atomic.AddInt64(&rs.numRecords, 1)
	atomic.AddInt64(&rs.dataSize, int64(len(data)))
	
	return nil
}

// UpdateRecord 更新记录
func (rs *BTreeRecordStore) UpdateRecord(ctx context.Context, recordId RecordId, data []byte) error {
	if recordId.IsNull() {
		return fmt.Errorf("RecordId 不能为空")
	}
	
	key, ok := recordId.AsBytes()
	if !ok {
		return fmt.Errorf("无法将 RecordId 转换为字节")
	}
	
	// 获取旧数据以更新统计
	oldData, exists := rs.tree.Get(key)
	if !exists {
		return fmt.Errorf("RecordId %s 不存在", recordId.String())
	}
	
	// 更新记录
	if err := rs.tree.Insert(key, data); err != nil {
		return fmt.Errorf("更新记录失败: %w", err)
	}
	
	// 更新统计
	atomic.AddInt64(&rs.dataSize, int64(len(data)-len(oldData)))
	
	return nil
}

// DeleteRecord 删除记录
func (rs *BTreeRecordStore) DeleteRecord(ctx context.Context, recordId RecordId) error {
	if recordId.IsNull() {
		return fmt.Errorf("RecordId 不能为空")
	}
	
	key, ok := recordId.AsBytes()
	if !ok {
		return fmt.Errorf("无法将 RecordId 转换为字节")
	}
	
	// 获取数据以更新统计
	data, exists := rs.tree.Get(key)
	if !exists {
		return fmt.Errorf("RecordId %s 不存在", recordId.String())
	}
	
	// 删除记录
	if err := rs.tree.Delete(key); err != nil {
		return fmt.Errorf("删除记录失败: %w", err)
	}
	
	// 更新统计
	atomic.AddInt64(&rs.numRecords, -1)
	atomic.AddInt64(&rs.dataSize, -int64(len(data)))
	
	return nil
}

// GetRecord 获取记录
func (rs *BTreeRecordStore) GetRecord(ctx context.Context, recordId RecordId) ([]byte, error) {
	if recordId.IsNull() {
		return nil, fmt.Errorf("RecordId 不能为空")
	}
	
	key, ok := recordId.AsBytes()
	if !ok {
		return nil, fmt.Errorf("无法将 RecordId 转换为字节")
	}
	
	data, exists := rs.tree.Get(key)
	if !exists {
		return nil, fmt.Errorf("RecordId %s 不存在", recordId.String())
	}
	
	return data, nil
}

// Scan 扫描记录
func (rs *BTreeRecordStore) Scan(ctx context.Context, startId RecordId) (RecordCursor, error) {
	var startKey []byte
	
	if !startId.IsNull() {
		var ok bool
		startKey, ok = startId.AsBytes()
		if !ok {
			return nil, fmt.Errorf("无法将 RecordId 转换为字节")
		}
	} else {
		startKey = []byte{0} // 从最小值开始
	}
	
	// 执行范围查询
	keys, values, err := rs.tree.Range(startKey, nil)
	if err != nil {
		return nil, fmt.Errorf("扫描失败: %w", err)
	}
	
	return &btreeCursor{
		keys:   keys,
		values: values,
		index:  -1,
	}, nil
}

// NumRecords 返回记录数
func (rs *BTreeRecordStore) NumRecords() int64 {
	return atomic.LoadInt64(&rs.numRecords)
}

// DataSize 返回数据大小
func (rs *BTreeRecordStore) DataSize() int64 {
	return atomic.LoadInt64(&rs.dataSize)
}

// Truncate 清空所有记录
func (rs *BTreeRecordStore) Truncate(ctx context.Context) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	
	// 重新创建 B+Tree
	rs.tree = btree.NewBTree(128)
	
	// 重置统计
	atomic.StoreInt64(&rs.numRecords, 0)
	atomic.StoreInt64(&rs.dataSize, 0)
	
	return nil
}

// btreeCursor B+Tree 游标实现
type btreeCursor struct {
	keys   [][]byte
	values [][]byte
	index  int
}

func (c *btreeCursor) Next() bool {
	c.index++
	return c.index < len(c.keys)
}

func (c *btreeCursor) RecordId() RecordId {
	if c.index < 0 || c.index >= len(c.keys) {
		return NullRecordId()
	}
	return NewRecordIdFromBytes(c.keys[c.index])
}

func (c *btreeCursor) Data() []byte {
	if c.index < 0 || c.index >= len(c.values) {
		return nil
	}
	return c.values[c.index]
}

func (c *btreeCursor) Close() error {
	c.keys = nil
	c.values = nil
	return nil
}
