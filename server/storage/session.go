package storage

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// EngineSession 存储引擎会话
// 维护会话状态、事务上下文和资源管理
type EngineSession interface {
	// 会话生命周期
	Begin(ctx context.Context) error
	End(ctx context.Context) error
	
	// 获取 RecoveryUnit（事务管理）
	GetRecoveryUnit() RecoveryUnit
	
	// 会话标识
	GetSessionId() string
	
	// 事务操作
	BeginTransaction(ctx context.Context) error
	CommitTransaction(ctx context.Context) error
	RollbackTransaction(ctx context.Context) error
	
	// 状态查询
	IsActive() bool
	InTransaction() bool
}

// WiredTigerSession WiredTiger 风格的会话实现
type WiredTigerSession struct {
	mu sync.RWMutex
	
	// 会话标识
	sessionId string
	
	// RecoveryUnit（事务管理）
	recoveryUnit RecoveryUnit
	
	// 会话状态
	active       bool
	inTransaction bool
	
	// 创建时间
	createdAt time.Time
	
	// 关联的引擎
	engine KVEngine
}

// NewEngineSession 创建新的引擎会话
func NewEngineSession(sessionId string, engine KVEngine) EngineSession {
	return &WiredTigerSession{
		sessionId:     sessionId,
		recoveryUnit:  NewRecoveryUnit(),
		active:        false,
		inTransaction: false,
		createdAt:     time.Now(),
		engine:        engine,
	}
}

// Begin 开始会话
func (s *WiredTigerSession) Begin(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.active {
		return fmt.Errorf("会话 %s 已经处于活动状态", s.sessionId)
	}
	
	s.active = true
	return nil
}

// End 结束会话
func (s *WiredTigerSession) End(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.active {
		return nil
	}
	
	// 如果还在事务中，先回滚
	if s.inTransaction {
		if err := s.recoveryUnit.Rollback(ctx); err != nil {
			return fmt.Errorf("结束会话时回滚事务失败: %w", err)
		}
		s.inTransaction = false
	}
	
	s.active = false
	return nil
}

// GetRecoveryUnit 获取 RecoveryUnit
func (s *WiredTigerSession) GetRecoveryUnit() RecoveryUnit {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.recoveryUnit
}

// GetSessionId 获取会话 ID
func (s *WiredTigerSession) GetSessionId() string {
	return s.sessionId
}

// BeginTransaction 开始事务
func (s *WiredTigerSession) BeginTransaction(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.active {
		return fmt.Errorf("会话 %s 未激活", s.sessionId)
	}
	
	if s.inTransaction {
		return fmt.Errorf("会话 %s 已经在事务中", s.sessionId)
	}
	
	if err := s.recoveryUnit.BeginTransaction(ctx); err != nil {
		return err
	}
	
	s.inTransaction = true
	return nil
}

// CommitTransaction 提交事务
func (s *WiredTigerSession) CommitTransaction(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.inTransaction {
		return fmt.Errorf("会话 %s 没有活动的事务", s.sessionId)
	}
	
	if err := s.recoveryUnit.Commit(ctx); err != nil {
		return err
	}
	
	s.inTransaction = false
	return nil
}

// RollbackTransaction 回滚事务
func (s *WiredTigerSession) RollbackTransaction(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.inTransaction {
		return fmt.Errorf("会话 %s 没有活动的事务", s.sessionId)
	}
	
	if err := s.recoveryUnit.Rollback(ctx); err != nil {
		return err
	}
	
	s.inTransaction = false
	return nil
}

// IsActive 检查会话是否活动
func (s *WiredTigerSession) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.active
}

// InTransaction 检查是否在事务中
func (s *WiredTigerSession) InTransaction() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.inTransaction
}
