package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/zhukovaskychina/xmongodb/cmd"
	"github.com/zhukovaskychina/xmongodb/config"
	"github.com/zhukovaskychina/xmongodb/logger"
	"github.com/zhukovaskychina/xmongodb/server"
)

var (
	configPath = flag.String("configPath", "./mongodb.conf", "配置文件路径")
	initialize = flag.Bool("initialize", false, "初始化数据库")
	debug      = flag.Bool("debug", false, "调试模式")
	version    = flag.Bool("version", false, "显示版本信息")
	showHelp   = flag.Bool("help", false, "显示帮助信息")
)

const (
	Version = "1.0.0"
	Build   = "dev"
)

func main() {
	flag.Parse()

	if *showHelp {
		printUsage()
		return
	}

	if *version {
		fmt.Printf("XMongoDB Server %s (Build: %s)\n", Version, Build)
		return
	}

	// 加载配置
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化日志
	logger.Init(cfg.Logger)

	// 如果是初始化模式
	if *initialize {
		if err := cmd.InitializeDatabase(cfg); err != nil {
			log.Fatalf("初始化数据库失败: %v", err)
		}
		fmt.Println("数据库初始化完成")
		return
	}

	// 创建并启动服务器
	srv := server.NewMongoDBServer(cfg)

	// 启动服务器
	if err := srv.Start(); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}

	// 等待信号
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	// 优雅关闭
	log.Println("正在关闭服务器...")
	if err := srv.Stop(); err != nil {
		log.Printf("关闭服务器时出错: %v", err)
	}
	log.Println("服务器已关闭")
}

func printUsage() {
	fmt.Printf(`XMongoDB Server %s - 基于 Go 实现的 MongoDB 兼容数据库

使用方法:
  %s [选项]

选项:
  -configPath string
        配置文件路径 (默认 "./mongodb.conf")
  -initialize
        初始化数据库
  -debug
        启用调试模式
  -version
        显示版本信息
  -help
        显示此帮助信息

示例:
  # 使用默认配置启动
  %s

  # 指定配置文件启动
  %s -configPath=./my-mongodb.conf

  # 初始化数据库
  %s -configPath=./my-mongodb.conf -initialize

  # 调试模式启动
  %s -configPath=./my-mongodb.conf -debug

`, Version, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}
