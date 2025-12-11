package storage

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RecoveryUnit 事务和快照抽象层
// 负责管理事务生命周期、快照隔离和回滚恢复
type RecoveryUnit interface {
	// 事务控制
	BeginTransaction(ctx context.Context) error
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
	
	// 快照和时间戳管理
	GetReadTimestamp() time.Time
	SetCommitTimestamp(ts time.Time) error
	
	// MVCC 历史存储（预留接口）
	PrepareForHistoryStore(oldValue []byte) error
	
	// 状态查询
	IsActive() bool
	IsCommitted() bool
	IsAborted() bool
	
	// 变更跟踪
	RegisterChange(change Change) error
}

// Change 表示一个可回滚的变更操作
type Change interface {
	Rollback() error
	Commit() error
}

// TransactionState 事务状态
type TransactionState int

const (
	TxnStateInactive TransactionState = iota
	TxnStateActive
	TxnStateCommitted
	TxnStateAborted
)

// WiredTigerRecoveryUnit WiredTiger 风格的 RecoveryUnit 实现
type WiredTigerRecoveryUnit struct {
	mu sync.RWMutex
	
	// 事务状态
	state TransactionState
	
	// 时间戳管理
	readTimestamp   time.Time
	commitTimestamp time.Time
	
	// 变更日志（用于回滚）
	changes []Change
	
	// 快照数据（简化版本）
	snapshot map[string][]byte
}

// NewRecoveryUnit 创建新的 RecoveryUnit
func NewRecoveryUnit() RecoveryUnit {
	return &WiredTigerRecoveryUnit{
		state:    TxnStateInactive,
		changes:  make([]Change, 0),
		snapshot: make(map[string][]byte),
	}
}

// BeginTransaction 开始事务
func (ru *WiredTigerRecoveryUnit) BeginTransaction(ctx context.Context) error {
	ru.mu.Lock()
	defer ru.mu.Unlock()
	
	if ru.state == TxnStateActive {
		return fmt.Errorf("事务已经处于活动状态")
	}
	
	// 重置状态
	ru.state = TxnStateActive
	ru.readTimestamp = time.Now()
	ru.changes = make([]Change, 0)
	ru.snapshot = make(map[string][]byte)
	
	return nil
}

// Commit 提交事务
func (ru *WiredTigerRecoveryUnit) Commit(ctx context.Context) error {
	ru.mu.Lock()
	defer ru.mu.Unlock()
	
	if ru.state != TxnStateActive {
		return fmt.Errorf("没有活动的事务可以提交")
	}
	
	// 设置提交时间戳
	if ru.commitTimestamp.IsZero() {
		ru.commitTimestamp = time.Now()
	}
	
	// 提交所有变更
	for _, change := range ru.changes {
		if err := change.Commit(); err != nil {
			// 提交失败，尝试回滚
			ru.state = TxnStateAborted
			return fmt.Errorf("提交变更失败: %w", err)
		}
	}
	
	ru.state = TxnStateCommitted
	ru.changes = nil
	
	return nil
}

// Rollback 回滚事务
func (ru *WiredTigerRecoveryUnit) Rollback(ctx context.Context) error {
	ru.mu.Lock()
	defer ru.mu.Unlock()
	
	if ru.state != TxnStateActive {
		return fmt.Errorf("没有活动的事务可以回滚")
	}
	
	// 逆序回滚所有变更
	for i := len(ru.changes) - 1; i >= 0; i-- {
		if err := ru.changes[i].Rollback(); err != nil {
			return fmt.Errorf("回滚变更失败: %w", err)
		}
	}
	
	ru.state = TxnStateAborted
	ru.changes = nil
	ru.snapshot = make(map[string][]byte)
	
	return nil
}

// GetReadTimestamp 获取读时间戳
func (ru *WiredTigerRecoveryUnit) GetReadTimestamp() time.Time {
	ru.mu.RLock()
	defer ru.mu.RUnlock()
	return ru.readTimestamp
}

// SetCommitTimestamp 设置提交时间戳
func (ru *WiredTigerRecoveryUnit) SetCommitTimestamp(ts time.Time) error {
	ru.mu.Lock()
	defer ru.mu.Unlock()
	
	if ru.state != TxnStateActive {
		return fmt.Errorf("只能在活动事务中设置提交时间戳")
	}
	
	ru.commitTimestamp = ts
	return nil
}

// PrepareForHistoryStore 为历史存储准备数据（预留接口，暂时为空实现）
func (ru *WiredTigerRecoveryUnit) PrepareForHistoryStore(oldValue []byte) error {
	// TODO: 实现 MVCC 历史版本存储
	// 在完整的 MVCC 实现中，这里会将旧版本数据保存到历史存储中
	// 以支持多版本并发控制和时间点查询
	return nil
}

// IsActive 检查事务是否活动
func (ru *WiredTigerRecoveryUnit) IsActive() bool {
	ru.mu.RLock()
	defer ru.mu.RUnlock()
	return ru.state == TxnStateActive
}

// IsCommitted 检查事务是否已提交
func (ru *WiredTigerRecoveryUnit) IsCommitted() bool {
	ru.mu.RLock()
	defer ru.mu.RUnlock()
	return ru.state == TxnStateCommitted
}

// IsAborted 检查事务是否已中止
func (ru *WiredTigerRecoveryUnit) IsAborted() bool {
	ru.mu.RLock()
	defer ru.mu.RUnlock()
	return ru.state == TxnStateAborted
}

// RegisterChange 注册一个可回滚的变更
func (ru *WiredTigerRecoveryUnit) RegisterChange(change Change) error {
	ru.mu.Lock()
	defer ru.mu.Unlock()
	
	if ru.state != TxnStateActive {
		return fmt.Errorf("只能在活动事务中注册变更")
	}
	
	ru.changes = append(ru.changes, change)
	return nil
}

// SimpleChange 简单的变更实现
type SimpleChange struct {
	commitFunc   func() error
	rollbackFunc func() error
}

// NewSimpleChange 创建简单变更
func NewSimpleChange(commitFunc, rollbackFunc func() error) Change {
	return &SimpleChange{
		commitFunc:   commitFunc,
		rollbackFunc: rollbackFunc,
	}
}

func (c *SimpleChange) Commit() error {
	if c.commitFunc != nil {
		return c.commitFunc()
	}
	return nil
}

func (c *SimpleChange) Rollback() error {
	if c.rollbackFunc != nil {
		return c.rollbackFunc()
	}
	return nil
}
