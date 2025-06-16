package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zhukovaskychina/xmongodb/config"
)

// InitializeDatabase 初始化数据库
func InitializeDatabase(cfg *config.Config) error {
	fmt.Println("开始初始化 XMongoDB 数据库...")

	// 创建数据目录
	if err := createDirectories(cfg); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 初始化存储引擎
	if err := initializeStorage(cfg); err != nil {
		return fmt.Errorf("初始化存储引擎失败: %w", err)
	}

	// 创建默认数据库和集合
	if err := createDefaultDatabase(cfg); err != nil {
		return fmt.Errorf("创建默认数据库失败: %w", err)
	}

	fmt.Println("数据库初始化完成!")
	return nil
}

// createDirectories 创建必要的目录
func createDirectories(cfg *config.Config) error {
	dirs := []string{
		cfg.Server.DataDir,
		cfg.Storage.DirectoryForDB,
		filepath.Join(cfg.Server.DataDir, "journal"),
		filepath.Join(cfg.Server.DataDir, "logs"),
		filepath.Join(cfg.Server.DataDir, "tmp"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %w", dir, err)
		}
		fmt.Printf("✓ 创建目录: %s\n", dir)
	}

	return nil
}

// initializeStorage 初始化存储引擎
func initializeStorage(cfg *config.Config) error {
	fmt.Printf("✓ 初始化存储引擎: %s\n", cfg.Storage.Engine)

	// 这里可以添加存储引擎的初始化逻辑
	// 例如创建WiredTiger的配置文件等

	return nil
}

// createDefaultDatabase 创建默认数据库和集合
func createDefaultDatabase(cfg *config.Config) error {
	fmt.Println("✓ 创建默认数据库结构")

	// 创建 admin 数据库目录
	adminDBPath := filepath.Join(cfg.Storage.DirectoryForDB, "admin")
	if err := os.MkdirAll(adminDBPath, 0755); err != nil {
		return fmt.Errorf("创建 admin 数据库目录失败: %w", err)
	}

	// 创建 local 数据库目录 (用于副本集oplog等)
	localDBPath := filepath.Join(cfg.Storage.DirectoryForDB, "local")
	if err := os.MkdirAll(localDBPath, 0755); err != nil {
		return fmt.Errorf("创建 local 数据库目录失败: %w", err)
	}

	// 创建 config 数据库目录 (用于分片配置)
	configDBPath := filepath.Join(cfg.Storage.DirectoryForDB, "config")
	if err := os.MkdirAll(configDBPath, 0755); err != nil {
		return fmt.Errorf("创建 config 数据库目录失败: %w", err)
	}

	fmt.Println("✓ 默认数据库结构创建完成")
	return nil
}
