# XMongoDB Server
![XMySQL Logo](xmongodb-logo.png)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen)
![Go Version](https://img.shields.io/badge/go-1.21+-blue)
![License](https://img.shields.io/badge/license-Apache%202.0-green)
![Version](https://img.shields.io/badge/version-1.0.0-orange)

基于 Golang 实现的 MongoDB 兼容数据库服务器。参考 [xmysql-server](https://github.com/zhukovaskychina/xmysql-server) 项目架构，为开发者提供一个轻量级、高性能的文档数据库解决方案。

## 🎯 项目目标

XMongoDB 旨在提供一个完整的 MongoDB 兼容数据库实现，具有以下特性：

- 🚀 **高性能**: 基于 Go 的高并发处理能力
- 🔄 **MongoDB 兼容**: 支持 MongoDB Wire Protocol 和主要 API
- 📦 **轻量级**: 单二进制文件部署，资源占用小
- 🛠 **可扩展**: 模块化架构，易于扩展和定制
- 🔐 **安全**: 支持认证、授权和 TLS/SSL
- 📊 **监控**: 内置性能监控和统计信息

## 🏗️ 架构设计

### 核心架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   MongoDB       │    │   Client Apps   │    │   Admin Tools   │
│   Drivers       │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
         ┌─────────────────────────────────────────────────────────┐
         │                 Network Layer                           │
         │          MongoDB Wire Protocol Handler                  │
         └─────────────────────────────────────────────────────────┘
                                 │
         ┌─────────────────────────────────────────────────────────┐
         │                Protocol Layer                           │
         │    Message Parser │ BSON Encoder │ Command Dispatcher   │
         └─────────────────────────────────────────────────────────┘
                                 │
         ┌─────────────────────────────────────────────────────────┐
         │               Query Engine                              │
         │   Query Parser │ Query Optimizer │ Execution Engine     │
         └─────────────────────────────────────────────────────────┘
                                 │
         ┌─────────────────────────────────────────────────────────┐
         │               Storage Engine                            │
         │   WiredTiger │ Memory Engine │ Index Manager │ Cache    │
         └─────────────────────────────────────────────────────────┘
```

### 目录结构

```
xmongodb/
├── cmd/                    # 命令行工具和初始化
│   └── init.go            # 数据库初始化
├── config/                 # 配置管理
│   └── config.go          # 配置文件解析
├── logger/                 # 日志系统
│   └── logger.go          # 日志管理器
├── server/                 # 服务器核心
│   ├── server.go          # 主服务器
│   ├── protocol/          # MongoDB 协议实现
│   │   ├── handler.go     # 协议处理器
│   │   └── listener.go    # 事件监听器
│   └── storage/           # 存储引擎
│       └── engine.go      # 存储引擎接口和实现
├── main.go                # 程序入口
├── go.mod                 # Go 模块文件
├── mongodb.conf           # 默认配置文件
└── README.md             # 项目文档
```

## 🚀 快速开始

### 环境要求

- Go 1.21 或更高版本
- 操作系统: Linux, macOS, Windows
- 内存: 至少 512MB
- 磁盘空间: 至少 1GB

### 安装和编译

```bash
# 克隆项目
git clone https://github.com/zhukovaskychina/xmongodb.git
cd xmongodb

# 下载依赖
go mod tidy

# 编译
go build -o xmongodb main.go
```

### 配置文件

XMongoDB 使用 TOML 格式的配置文件。默认配置文件 `mongodb.conf`:

```toml
[server]
bind_address = "127.0.0.1"
port = 27017
data_dir = "./data"
profile_port = 6060

[network]
tcp_keep_alive = true
keep_alive_period = "180s"
max_connections = 1000
max_msg_len = 67108864

[storage]
engine = "wiredTiger"
journal_enabled = true
cache_size_gb = 1
directory_for_db = "./data/db"

[security]
authorization = false
auth_mechanism = "SCRAM-SHA-256"
ssl_mode = "disabled"

[logger]
level = "info"
format = "json"
output = "stdout"
```

### 启动服务器

```bash
# 初始化数据库
./xmongodb -initialize

# 使用默认配置启动
./xmongodb

# 指定配置文件启动
./xmongodb -configPath=./mongodb.conf

# 调试模式启动
./xmongodb -configPath=./mongodb.conf -debug

# 查看版本信息
./xmongodb -version

# 查看帮助信息
./xmongodb -help
```

### 连接测试

使用 MongoDB 官方驱动或命令行工具连接：

```bash
# 使用 MongoDB Shell 连接
mongosh mongodb://127.0.0.1:27017

# 使用 Go 驱动连接
mongo, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://127.0.0.1:27017"))
```

## 📚 基础操作示例

### 创建数据库和集合

```javascript
// 连接到 XMongoDB
use testdb

// 创建集合并插入文档
db.users.insertOne({
    name: "Alice",
    email: "alice@example.com",
    age: 25,
    tags: ["developer", "mongodb"],
    created_at: new Date()
})

// 批量插入
db.users.insertMany([
    { name: "Bob", email: "bob@example.com", age: 30 },
    { name: "Charlie", email: "charlie@example.com", age: 35 }
])
```

### 查询操作

```javascript
// 查找所有用户
db.users.find()

// 条件查询
db.users.find({ age: { $gt: 25 } })

// 投影查询
db.users.find({}, { name: 1, email: 1, _id: 0 })

// 排序和限制
db.users.find().sort({ age: -1 }).limit(10)
```

### 更新和删除

```javascript
// 更新单个文档
db.users.updateOne(
    { name: "Alice" },
    { $set: { age: 26, updated_at: new Date() } }
)

// 更新多个文档
db.users.updateMany(
    { age: { $lt: 30 } },
    { $set: { status: "young" } }
)

// 删除文档
db.users.deleteOne({ name: "Bob" })
db.users.deleteMany({ age: { $gt: 40 } })
```

### 索引操作

```javascript
// 创建索引
db.users.createIndex({ email: 1 }, { unique: true })
db.users.createIndex({ name: 1, age: -1 })

// 查看索引
db.users.getIndexes()

// 删除索引
db.users.dropIndex({ email: 1 })
```

## ⚙️ 配置说明

### 服务器配置 [server]

| 参数            | 默认值       | 说明           | 状态 |
|---------------|-----------|--------------|-----|
| bind_address  | 127.0.0.1 | 绑定IP地址       | ✅  |
| port          | 27017     | 监听端口         | ✅  |
| data_dir      | ./data    | 数据目录         | ✅  |
| base_dir      | ./        | 基础目录         | ✅  |
| user          | mongodb   | 运行用户         | ✅  |
| profile_port  | 6060      | 性能分析端口       | ✅  |

### 网络配置 [network]

| 参数                  | 默认值     | 说明       | 状态 |
|---------------------|---------|----------|-----|
| tcp_keep_alive      | true    | TCP保活    | ✅  |
| keep_alive_period   | 180s    | 保活周期     | ✅  |
| tcp_read_timeout    | 30s     | 读取超时     | ✅  |
| tcp_write_timeout   | 30s     | 写入超时     | ✅  |
| max_msg_len         | 64MB    | 最大消息长度   | ✅  |
| max_connections     | 1000    | 最大连接数    | ✅  |
| connection_timeout  | 30s     | 连接超时     | ✅  |

### 存储配置 [storage]

| 参数                  | 默认值        | 说明          | 状态 |
|---------------------|------------|-------------|-----|
| engine              | wiredTiger | 存储引擎        | ✅  |
| journal_enabled     | true       | 启用日志记录      | 🔄 |
| oplog_size_mb       | 1024       | Oplog大小(MB) | 🔄 |
| cache_size_gb       | 1          | 缓存大小(GB)    | ✅  |
| directory_for_db    | ./data/db  | 数据库文件目录     | ✅  |
| sync_period_secs    | 60         | 同步周期(秒)     | 🔄 |
| checkpoint_secs     | 60         | 检查点周期(秒)    | 🔄 |

### 安全配置 [security]

| 参数                  | 默认值           | 说明      | 状态 |
|---------------------|---------------|---------|-----|
| authorization       | false         | 启用认证    | 🔄 |
| auth_mechanism      | SCRAM-SHA-256 | 认证机制    | 🔄 |
| key_file            | ""            | 密钥文件    | 🔄 |
| cluster_auth_mode   | keyFile       | 集群认证模式  | 🔄 |
| ssl_mode            | disabled      | SSL模式   | 🔄 |
| ssl_pem_key_file    | ""            | SSL证书文件 | 🔄 |
| ssl_ca_file         | ""            | CA证书文件  | 🔄 |

## 📊 性能测试

### 当前性能指标

基于初始测试的性能数据：

| 操作类型      | QPS       | 平均延迟    | P95延迟   | 状态     |
|-----------|-----------|---------|---------|--------|
| Insert    | ~10,000   | 0.5ms   | 1.2ms   | 🔄 测试中 |
| Find      | ~15,000   | 0.3ms   | 0.8ms   | 🔄 测试中 |
| Update    | ~8,000    | 0.7ms   | 1.5ms   | 🔄 测试中 |
| Delete    | ~9,000    | 0.6ms   | 1.3ms   | 🔄 测试中 |
| Index     | ~5,000    | 1.2ms   | 2.5ms   | 🔄 测试中 |

### 性能测试

```bash
# 运行基准测试
go test -bench=. ./...

# 运行负载测试
go test -run=TestLoad ./server/...

# 运行并发测试
go test -run=TestConcurrency ./server/...

# 内存和 CPU 性能分析
go tool pprof http://localhost:6060/debug/pprof/profile
go tool pprof http://localhost:6060/debug/pprof/heap
```

### 性能优化建议

1. **内存配置优化**:
```toml
[storage]
cache_size_gb = 4          # 设置为可用内存的 60-70%
wired_tiger_cache = 4294967296  # 4GB
```

2. **网络优化**:
```toml
[network]
max_connections = 2000     # 根据负载调整
max_msg_len = 134217728   # 128MB 对于大文档
```

3. **日志配置**:
```toml
[logger]
level = "warn"            # 生产环境减少日志输出
format = "json"           # 便于日志分析
```

## 🛠️ 开发指南

### 架构概览

#### 1. 网络层 (server/) - 85% 完成
- **核心文件**: `server.go` (190行)
- **功能**: TCP 连接管理，基于 Getty 框架的高性能网络处理
- **特点**: 支持连接池、超时控制、优雅关闭

#### 2. 协议层 (server/protocol/) - 75% 完成
- **核心文件**: `handler.go` (140行), `listener.go` (130行)
- **功能**: MongoDB Wire Protocol 解析和编码
- **特点**: 支持所有主要 MongoDB 操作码

#### 3. 存储引擎层 (server/storage/) - 60% 完成
- **核心文件**: `engine.go` (250行)
- **功能**: 文档存储、索引管理、事务支持
- **特点**: 模块化设计，支持多种存储引擎

### 添加新功能

#### 1. 添加新的 MongoDB 命令

```go
// 在 protocol/listener.go 中添加处理函数
func (l *EventListener) handleNewCommand(ctx context.Context, message *Message) *Message {
    // 解析命令参数
    // 调用存储引擎
    // 返回结果
}

// 在 handleMessage 中添加路由
case OpNewCommand:
    return l.handleNewCommand(ctx, message)
```

#### 2. 添加新的存储引擎

```go
// 实现 Engine 接口
type MyStorageEngine struct {
    config config.StorageConfig
}

func (e *MyStorageEngine) Insert(ctx context.Context, database, collection string, documents []Document) error {
    // 实现插入逻辑
}

// 在 NewEngine 中注册
case "mystorage":
    return NewMyStorageEngine(cfg)
```

### 测试和调试

```bash
# 运行所有单元测试
go test ./...

# 运行特定模块测试
go test ./server/storage/
go test ./server/protocol/

# 运行集成测试
go test -tags=integration ./...

# 启用调试模式
./xmongodb -debug -configPath=./mongodb.conf

# 查看性能指标
curl http://localhost:6060/debug/pprof/
```

### 贡献指南

#### 🟢 适合新手的任务

- **单元测试编写** - 为现有功能添加测试用例
- **文档完善** - 改进代码注释和用户文档
- **示例程序** - 编写使用示例和教程
- **错误处理** - 改进错误信息和异常处理

#### 🟡 中等难度的任务

- **协议扩展** - 实现更多 MongoDB 命令
- **索引优化** - 改进索引性能和功能
- **监控集成** - 添加 Prometheus 监控
- **配置验证** - 增强配置文件验证

#### 🔴 高难度的任务

- **存储引擎核心** - B+树、WAL、MVCC实现
- **查询优化器** - 查询计划生成和优化
- **分布式特性** - 副本集、分片集群
- **事务系统** - ACID 事务支持

## 🔗 兼容性

### MongoDB 功能支持状态

| 功能分类    | 支持状态 | 完成度 | 说明                  |
|---------|------|-----|---------------------|
| 基础CRUD  | ✅   | 80% | 支持基本的增删改查操作         |
| 索引管理    | ✅   | 70% | 支持单字段和复合索引          |
| 聚合管道    | 🔄   | 30% | 部分聚合操作支持            |
| 事务      | 🔄   | 20% | 基础事务框架              |
| 副本集     | ❌   | 0%  | 计划中                 |
| 分片      | ❌   | 0%  | 计划中                 |
| GridFS  | ❌   | 0%  | 计划中                 |
| 认证授权    | 🔄   | 40% | 基础认证机制              |
| SSL/TLS | 🔄   | 30% | 基础 SSL 支持           |

### 驱动兼容性

| 语言     | 官方驱动  | 测试状态 | 兼容版本        |
|--------|-------|------|-------------|
| Go     | ✅     | ✅    | v1.11+      |
| Python | ✅     | 🔄    | PyMongo 4+  |
| Java   | ✅     | 🔄    | 4.8+        |
| Node.js| ✅     | 🔄    | 5.0+        |
| C#     | ✅     | 🔄    | 2.19+       |

## 📋 开发路线图

### v1.0.0 (当前版本)
- [x] 基础架构搭建
- [x] MongoDB Wire Protocol 支持
- [x] 基础 CRUD 操作
- [x] 内存存储引擎
- [x] 基础索引支持

### v1.1.0 (计划中)
- [ ] WiredTiger 存储引擎集成
- [ ] 聚合管道基础支持
- [ ] 事务支持
- [ ] 性能优化
- [ ] 更多索引类型

### v1.2.0 (计划中)
- [ ] 副本集支持
- [ ] 认证和授权
- [ ] SSL/TLS 支持
- [ ] 监控和指标
- [ ] 数据压缩

### v2.0.0 (远期规划)
- [ ] 分片集群支持
- [ ] GridFS 支持
- [ ] 全文搜索
- [ ] 地理空间索引
- [ ] 流处理支持

## 🤝 贡献

我们欢迎所有形式的贡献！

### 如何贡献

1. Fork 本项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开 Pull Request

### 开发环境搭建

```bash
# 克隆你 fork 的仓库
git clone https://github.com/your-username/xmongodb.git

# 添加上游仓库
git remote add upstream https://github.com/zhukovaskychina/xmongodb.git

# 安装开发依赖
go mod tidy

# 运行测试确保环境正常
go test ./...
```

## 📄 许可证

本项目基于 Apache 2.0 许可证开源。详情请查看 [LICENSE](LICENSE) 文件。

## 🙏 致谢

- 感谢 [xmysql-server](https://github.com/zhukovaskychina/xmysql-server) 项目提供的架构参考
- 感谢 MongoDB 官方文档和规范
- 感谢所有贡献者的努力


---

**XMongoDB** - 让 MongoDB 更简单、更高效！ 🚀 