package storage

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	
	"github.com/zhukovaskychina/xmongodb/server/storage/btree"
)

// SortedDataInterface 索引数据接口
// 用于管理有序的索引数据，支持范围查询
type SortedDataInterface interface {
	// 插入索引条目
	Insert(ctx context.Context, key []byte, recordId RecordId) error
	
	// 删除索引条目
	Remove(ctx context.Context, key []byte, recordId RecordId) error
	
	// 查找精确匹配的记录
	Seek(ctx context.Context, key []byte) (IndexCursor, error)
	
	// 范围查询
	SeekRange(ctx context.Context, startKey, endKey []byte) (IndexCursor, error)
	
	// 统计信息
	NumEntries() int64
	IsEmpty() bool
	
	// 清空索引
	Clear(ctx context.Context) error
}

// IndexCursor 索引游标
type IndexCursor interface {
	Next() bool
	Key() []byte
	RecordId() RecordId
	Close() error
}

// IndexKeyEntry 索引键条目
// 组合索引键和 RecordId
type IndexKeyEntry struct {
	Key      []byte
	RecordId RecordId
}

// BTreeIndex 基于 B+Tree 的索引实现
type BTreeIndex struct {
	mu sync.RWMutex
	
	// B+Tree 存储
	// Key: indexKey + recordId (组合键确保唯一性)
	// Value: recordId (冗余存储便于查询)
	tree *btree.BTree
	
	// 索引配置
	name      string
	unique    bool
	numEntries int64
}

// NewSortedDataInterface 创建新的索引
func NewSortedDataInterface(name string, unique bool) SortedDataInterface {
	return &BTreeIndex{
		tree:   btree.NewBTree(128),
		name:   name,
		unique: unique,
	}
}

// Insert 插入索引条目
func (idx *BTreeIndex) Insert(ctx context.Context, key []byte, recordId RecordId) error {
	if len(key) == 0 {
		return fmt.Errorf("索引键不能为空")
	}
	
	if recordId.IsNull() {
		return fmt.Errorf("RecordId 不能为空")
	}
	
	// 如果是唯一索引，检查是否已存在
	if idx.unique {
		if exists, err := idx.keyExists(key); err != nil {
			return err
		} else if exists {
			return fmt.Errorf("唯一索引约束违反: 键 %x 已存在", key)
		}
	}
	
	// 组合键: indexKey + recordId
	compositeKey := idx.makeCompositeKey(key, recordId)
	
	// RecordId 作为值
	recordIdBytes, _ := recordId.AsBytes()
	
	// 插入到 B+Tree
	if err := idx.tree.Insert(compositeKey, recordIdBytes); err != nil {
		return fmt.Errorf("插入索引失败: %w", err)
	}
	
	idx.mu.Lock()
	idx.numEntries++
	idx.mu.Unlock()
	
	return nil
}

// Remove 删除索引条目
func (idx *BTreeIndex) Remove(ctx context.Context, key []byte, recordId RecordId) error {
	if len(key) == 0 {
		return fmt.Errorf("索引键不能为空")
	}
	
	if recordId.IsNull() {
		return fmt.Errorf("RecordId 不能为空")
	}
	
	// 组合键
	compositeKey := idx.makeCompositeKey(key, recordId)
	
	// 从 B+Tree 删除
	if err := idx.tree.Delete(compositeKey); err != nil {
		return fmt.Errorf("删除索引失败: %w", err)
	}
	
	idx.mu.Lock()
	idx.numEntries--
	idx.mu.Unlock()
	
	return nil
}

// Seek 查找精确匹配的记录
func (idx *BTreeIndex) Seek(ctx context.Context, key []byte) (IndexCursor, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("索引键不能为空")
	}
	
	// 构造范围查询的起始键和结束键
	startKey := idx.makeCompositeKey(key, NullRecordId())
	endKey := idx.makeNextKey(key)
	
	// 执行范围查询
	keys, values, err := idx.tree.Range(startKey, endKey)
	if err != nil {
		return nil, fmt.Errorf("查找失败: %w", err)
	}
	
	return &btreeIndexCursor{
		keys:   keys,
		values: values,
		index:  -1,
	}, nil
}

// SeekRange 范围查询
func (idx *BTreeIndex) SeekRange(ctx context.Context, startKey, endKey []byte) (IndexCursor, error) {
	var start, end []byte
	
	if startKey != nil {
		start = idx.makeCompositeKey(startKey, NullRecordId())
	}
	
	if endKey != nil {
		end = idx.makeCompositeKey(endKey, NullRecordId())
	}
	
	// 执行范围查询
	keys, values, err := idx.tree.Range(start, end)
	if err != nil {
		return nil, fmt.Errorf("范围查询失败: %w", err)
	}
	
	return &btreeIndexCursor{
		keys:   keys,
		values: values,
		index:  -1,
	}, nil
}

// NumEntries 返回索引条目数
func (idx *BTreeIndex) NumEntries() int64 {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.numEntries
}

// IsEmpty 检查索引是否为空
func (idx *BTreeIndex) IsEmpty() bool {
	return idx.NumEntries() == 0
}

// Clear 清空索引
func (idx *BTreeIndex) Clear(ctx context.Context) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	
	// 重新创建 B+Tree
	idx.tree = btree.NewBTree(128)
	idx.numEntries = 0
	
	return nil
}

// makeCompositeKey 创建组合键
// 格式: [keyLen(4字节)][key][recordId]
func (idx *BTreeIndex) makeCompositeKey(key []byte, recordId RecordId) []byte {
	recordIdBytes, _ := recordId.AsBytes()
	
	// 计算总长度
	totalLen := 4 + len(key) + len(recordIdBytes)
	composite := make([]byte, totalLen)
	
	// 写入键长度（大端序）
	composite[0] = byte(len(key) >> 24)
	composite[1] = byte(len(key) >> 16)
	composite[2] = byte(len(key) >> 8)
	composite[3] = byte(len(key))
	
	// 写入键
	copy(composite[4:], key)
	
	// 写入 RecordId
	copy(composite[4+len(key):], recordIdBytes)
	
	return composite
}

// parseCompositeKey 解析组合键
func (idx *BTreeIndex) parseCompositeKey(composite []byte) ([]byte, RecordId, error) {
	if len(composite) < 4 {
		return nil, NullRecordId(), fmt.Errorf("组合键太短")
	}
	
	// 读取键长度
	keyLen := int(composite[0])<<24 | int(composite[1])<<16 | int(composite[2])<<8 | int(composite[3])
	
	if len(composite) < 4+keyLen {
		return nil, NullRecordId(), fmt.Errorf("组合键格式错误")
	}
	
	// 提取键
	key := composite[4 : 4+keyLen]
	
	// 提取 RecordId
	recordIdBytes := composite[4+keyLen:]
	recordId := NewRecordIdFromBytes(recordIdBytes)
	
	return key, recordId, nil
}

// makeNextKey 创建下一个键（用于范围查询的上界）
func (idx *BTreeIndex) makeNextKey(key []byte) []byte {
	nextKey := make([]byte, len(key)+1)
	copy(nextKey, key)
	// 在末尾添加一个字节以表示"大于"
	nextKey[len(key)] = 0xFF
	return idx.makeCompositeKey(nextKey, NullRecordId())
}

// keyExists 检查键是否存在（用于唯一索引）
func (idx *BTreeIndex) keyExists(key []byte) (bool, error) {
	startKey := idx.makeCompositeKey(key, NullRecordId())
	endKey := idx.makeNextKey(key)
	
	keys, _, err := idx.tree.Range(startKey, endKey)
	if err != nil {
		return false, err
	}
	
	return len(keys) > 0, nil
}

// btreeIndexCursor B+Tree 索引游标实现
type btreeIndexCursor struct {
	keys   [][]byte
	values [][]byte
	index  int
}

func (c *btreeIndexCursor) Next() bool {
	c.index++
	return c.index < len(c.keys)
}

func (c *btreeIndexCursor) Key() []byte {
	if c.index < 0 || c.index >= len(c.keys) {
		return nil
	}
	
	// 解析组合键，提取索引键
	compositeKey := c.keys[c.index]
	if len(compositeKey) < 4 {
		return nil
	}
	
	keyLen := int(compositeKey[0])<<24 | int(compositeKey[1])<<16 | int(compositeKey[2])<<8 | int(compositeKey[3])
	if len(compositeKey) < 4+keyLen {
		return nil
	}
	
	return compositeKey[4 : 4+keyLen]
}

func (c *btreeIndexCursor) RecordId() RecordId {
	if c.index < 0 || c.index >= len(c.values) {
		return NullRecordId()
	}
	return NewRecordIdFromBytes(c.values[c.index])
}

func (c *btreeIndexCursor) Close() error {
	c.keys = nil
	c.values = nil
	return nil
}
