package btree

import (
	"bytes"
	"fmt"
	"sync"
)

// BTree B+树实现
// 用于存储有序的键值对，支持范围查询
type BTree struct {
	mu    sync.RWMutex
	root  *Node
	order int // B+树的阶数（每个节点最多的子节点数）
}

// Node B+树节点
type Node struct {
	isLeaf   bool
	keys     [][]byte  // 键列表
	values   [][]byte  // 值列表（仅叶子节点使用）
	children []*Node   // 子节点列表（仅内部节点使用）
	next     *Node     // 下一个叶子节点（仅叶子节点使用，用于范围查询）
	parent   *Node     // 父节点
}

// NewBTree 创建新的 B+树
func NewBTree(order int) *BTree {
	if order < 3 {
		order = 3 // 最小阶数为3
	}
	return &BTree{
		root:  newLeafNode(),
		order: order,
	}
}

// newLeafNode 创建新的叶子节点
func newLeafNode() *Node {
	return &Node{
		isLeaf: true,
		keys:   make([][]byte, 0),
		values: make([][]byte, 0),
	}
}

// newInternalNode 创建新的内部节点
func newInternalNode() *Node {
	return &Node{
		isLeaf:   false,
		keys:     make([][]byte, 0),
		children: make([]*Node, 0),
	}
}

// Insert 插入键值对
func (t *BTree) Insert(key, value []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	if len(key) == 0 {
		return fmt.Errorf("键不能为空")
	}
	
	// 复制键和值
	keyCopy := make([]byte, len(key))
	copy(keyCopy, key)
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)
	
	// 查找插入位置
	leaf := t.findLeaf(keyCopy)
	
	// 在叶子节点中插入
	t.insertIntoLeaf(leaf, keyCopy, valueCopy)
	
	// 检查是否需要分裂
	if len(leaf.keys) >= t.order {
		t.splitLeaf(leaf)
	}
	
	return nil
}

// Get 查找键对应的值
func (t *BTree) Get(key []byte) ([]byte, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	if len(key) == 0 {
		return nil, false
	}
	
	leaf := t.findLeaf(key)
	
	// 在叶子节点中查找
	for i, k := range leaf.keys {
		if bytes.Equal(k, key) {
			// 返回值的副本
			valueCopy := make([]byte, len(leaf.values[i]))
			copy(valueCopy, leaf.values[i])
			return valueCopy, true
		}
	}
	
	return nil, false
}

// Delete 删除键值对
func (t *BTree) Delete(key []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	if len(key) == 0 {
		return fmt.Errorf("键不能为空")
	}
	
	leaf := t.findLeaf(key)
	
	// 在叶子节点中查找并删除
	for i, k := range leaf.keys {
		if bytes.Equal(k, key) {
			leaf.keys = append(leaf.keys[:i], leaf.keys[i+1:]...)
			leaf.values = append(leaf.values[:i], leaf.values[i+1:]...)
			return nil
		}
	}
	
	return fmt.Errorf("键不存在")
}

// Range 范围查询
// 返回 [startKey, endKey) 范围内的所有键值对
func (t *BTree) Range(startKey, endKey []byte) ([][]byte, [][]byte, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	keys := make([][]byte, 0)
	values := make([][]byte, 0)
	
	// 找到起始叶子节点
	leaf := t.findLeaf(startKey)
	
	// 遍历叶子节点链表
	for leaf != nil {
		for i, k := range leaf.keys {
			// 检查是否在范围内
			if bytes.Compare(k, startKey) >= 0 {
				if endKey != nil && bytes.Compare(k, endKey) >= 0 {
					return keys, values, nil
				}
				
				// 复制键和值
				keyCopy := make([]byte, len(k))
				copy(keyCopy, k)
				valueCopy := make([]byte, len(leaf.values[i]))
				copy(valueCopy, leaf.values[i])
				
				keys = append(keys, keyCopy)
				values = append(values, valueCopy)
			}
		}
		
		leaf = leaf.next
	}
	
	return keys, values, nil
}

// findLeaf 查找包含指定键的叶子节点
func (t *BTree) findLeaf(key []byte) *Node {
	node := t.root
	
	for !node.isLeaf {
		// 在内部节点中查找
		i := 0
		for i < len(node.keys) && bytes.Compare(key, node.keys[i]) >= 0 {
			i++
		}
		node = node.children[i]
	}
	
	return node
}

// insertIntoLeaf 在叶子节点中插入键值对
func (t *BTree) insertIntoLeaf(leaf *Node, key, value []byte) {
	// 找到插入位置
	i := 0
	for i < len(leaf.keys) && bytes.Compare(key, leaf.keys[i]) > 0 {
		i++
	}
	
	// 检查是否已存在（更新）
	if i < len(leaf.keys) && bytes.Equal(key, leaf.keys[i]) {
		leaf.values[i] = value
		return
	}
	
	// 插入新的键值对
	leaf.keys = append(leaf.keys[:i], append([][]byte{key}, leaf.keys[i:]...)...)
	leaf.values = append(leaf.values[:i], append([][]byte{value}, leaf.values[i:]...)...)
}

// splitLeaf 分裂叶子节点
func (t *BTree) splitLeaf(leaf *Node) {
	mid := len(leaf.keys) / 2
	
	// 创建新的叶子节点
	newLeaf := newLeafNode()
	newLeaf.keys = append(newLeaf.keys, leaf.keys[mid:]...)
	newLeaf.values = append(newLeaf.values, leaf.values[mid:]...)
	newLeaf.next = leaf.next
	
	// 更新原叶子节点
	leaf.keys = leaf.keys[:mid]
	leaf.values = leaf.values[:mid]
	leaf.next = newLeaf
	
	// 提升中间键到父节点
	promoteKey := newLeaf.keys[0]
	
	if leaf.parent == nil {
		// 创建新的根节点
		newRoot := newInternalNode()
		newRoot.keys = append(newRoot.keys, promoteKey)
		newRoot.children = append(newRoot.children, leaf, newLeaf)
		leaf.parent = newRoot
		newLeaf.parent = newRoot
		t.root = newRoot
	} else {
		// 插入到现有父节点
		t.insertIntoParent(leaf.parent, promoteKey, newLeaf)
		newLeaf.parent = leaf.parent
	}
}

// insertIntoParent 在父节点中插入键和子节点
func (t *BTree) insertIntoParent(parent *Node, key []byte, rightChild *Node) {
	// 找到插入位置
	i := 0
	for i < len(parent.keys) && bytes.Compare(key, parent.keys[i]) > 0 {
		i++
	}
	
	// 插入键和子节点
	parent.keys = append(parent.keys[:i], append([][]byte{key}, parent.keys[i:]...)...)
	parent.children = append(parent.children[:i+1], append([]*Node{rightChild}, parent.children[i+1:]...)...)
	
	// 检查是否需要分裂父节点
	if len(parent.keys) >= t.order {
		t.splitInternal(parent)
	}
}

// splitInternal 分裂内部节点
func (t *BTree) splitInternal(node *Node) {
	mid := len(node.keys) / 2
	promoteKey := node.keys[mid]
	
	// 创建新的内部节点
	newNode := newInternalNode()
	newNode.keys = append(newNode.keys, node.keys[mid+1:]...)
	newNode.children = append(newNode.children, node.children[mid+1:]...)
	
	// 更新子节点的父指针
	for _, child := range newNode.children {
		child.parent = newNode
	}
	
	// 更新原节点
	node.keys = node.keys[:mid]
	node.children = node.children[:mid+1]
	
	if node.parent == nil {
		// 创建新的根节点
		newRoot := newInternalNode()
		newRoot.keys = append(newRoot.keys, promoteKey)
		newRoot.children = append(newRoot.children, node, newNode)
		node.parent = newRoot
		newNode.parent = newRoot
		t.root = newRoot
	} else {
		// 插入到现有父节点
		t.insertIntoParent(node.parent, promoteKey, newNode)
		newNode.parent = node.parent
	}
}

// Size 返回树中键值对的数量
func (t *BTree) Size() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	count := 0
	leaf := t.findFirstLeaf()
	for leaf != nil {
		count += len(leaf.keys)
		leaf = leaf.next
	}
	
	return count
}

// findFirstLeaf 找到第一个叶子节点
func (t *BTree) findFirstLeaf() *Node {
	node := t.root
	for !node.isLeaf {
		node = node.children[0]
	}
	return node
}
