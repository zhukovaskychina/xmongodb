package server

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/AlexStocks/getty"
	"github.com/zhukovaskychina/xmongodb/config"
	"github.com/zhukovaskychina/xmongodb/logger"
	"github.com/zhukovaskychina/xmongodb/server/protocol"
	"github.com/zhukovaskychina/xmongodb/server/storage"
)

// MongoDBServer MongoDB 服务器
type MongoDBServer struct {
	config        *config.Config
	tcpServer     getty.Server
	storageEngine storage.Engine
	mu            sync.RWMutex
	running       bool
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewMongoDBServer 创建新的 MongoDB 服务器
func NewMongoDBServer(cfg *config.Config) *MongoDBServer {
	ctx, cancel := context.WithCancel(context.Background())

	return &MongoDBServer{
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start 启动服务器
func (s *MongoDBServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("服务器已经在运行")
	}

	logger.Infof("启动 XMongoDB 服务器在 %s:%d", s.config.Server.BindAddress, s.config.Server.Port)

	// 初始化存储引擎
	var err error
	s.storageEngine, err = storage.NewEngine(s.config.Storage)
	if err != nil {
		return fmt.Errorf("初始化存储引擎失败: %w", err)
	}

	// 创建 TCP 服务器
	if err := s.startTCPServer(); err != nil {
		return fmt.Errorf("启动 TCP 服务器失败: %w", err)
	}

	s.running = true
	logger.Info("XMongoDB 服务器启动成功")
	return nil
}

// Stop 停止服务器
func (s *MongoDBServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	logger.Info("正在关闭 XMongoDB 服务器...")

	// 关闭 TCP 服务器
	if s.tcpServer != nil {
		s.tcpServer.Close()
	}

	// 关闭存储引擎
	if s.storageEngine != nil {
		if err := s.storageEngine.Close(); err != nil {
			logger.Errorf("关闭存储引擎失败: %v", err)
		}
	}

	// 取消上下文
	s.cancel()

	s.running = false
	logger.Info("XMongoDB 服务器已关闭")
	return nil
}

// startTCPServer 启动 TCP 服务器
func (s *MongoDBServer) startTCPServer() error {
	// Getty 服务器选项
	options := []getty.ServerOption{
		getty.WithLocalAddress(fmt.Sprintf("%s:%d", s.config.Server.BindAddress, s.config.Server.Port)),
	}

	// 创建 Getty 服务器
	s.tcpServer = getty.NewTCPServer(options...)

	// 设置事件处理器
	s.tcpServer.RunEventLoop(s.newSession)

	logger.Infof("TCP 服务器监听在 %s:%d", s.config.Server.BindAddress, s.config.Server.Port)

	go func() {
		select {
		case <-s.ctx.Done():
			return
		}
	}()

	return nil
}

// newSession 创建新的会话
func (s *MongoDBServer) newSession(session getty.Session) error {
	// 设置会话属性
	session.SetPkgHandler(protocol.NewPackageHandler())
	session.SetEventListener(protocol.NewEventListener(s.storageEngine))
	session.SetReadTimeout(30 * time.Second)
	session.SetWriteTimeout(30 * time.Second)
	session.SetCronPeriod(int(30 * time.Second.Nanoseconds() / 1e6))
	session.SetWaitTime(1 * time.Second)

	logger.Debugf("新会话建立: %s", session.RemoteAddr())
	return nil
}

// IsRunning 检查服务器是否在运行
func (s *MongoDBServer) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetConfig 获取服务器配置
func (s *MongoDBServer) GetConfig() *config.Config {
	return s.config
}

// GetStorageEngine 获取存储引擎
func (s *MongoDBServer) GetStorageEngine() storage.Engine {
	return s.storageEngine
}

// GetStats 获取服务器统计信息
func (s *MongoDBServer) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	stats["running"] = s.IsRunning()
	stats["bind_address"] = s.config.Server.BindAddress
	stats["port"] = s.config.Server.Port
	stats["storage_engine"] = s.config.Storage.Engine

	if s.storageEngine != nil {
		if storageStats := s.storageEngine.GetStats(); storageStats != nil {
			stats["storage"] = storageStats
		}
	}

	return stats
}

// validateConfig 验证配置
func (s *MongoDBServer) validateConfig() error {
	// 验证绑定地址
	if s.config.Server.BindAddress == "" {
		return fmt.Errorf("绑定地址不能为空")
	}

	// 验证端口
	if s.config.Server.Port <= 0 || s.config.Server.Port > 65535 {
		return fmt.Errorf("端口必须在 1-65535 范围内")
	}

	// 检查端口是否被占用
	addr := fmt.Sprintf("%s:%d", s.config.Server.BindAddress, s.config.Server.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("端口 %d 已被占用: %w", s.config.Server.Port, err)
	}
	listener.Close()

	return nil
}
