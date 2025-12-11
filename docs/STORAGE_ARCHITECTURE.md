# XMongoDB 存储引擎架构文档

## 概述

XMongoDB 存储引擎采用分层架构设计，从客户端请求到物理存储共分为以下几层：

```
Client Request (MongoDB Wire Protocol)
         ↓
   Protocol Layer (Getty + PackageHandler)
         ↓
    Engine API (WiredTigerEngine)
         ↓
  KVEngine (底层键值存储引擎)
         ↓
  ┌──────────────┬───────────────────┐
  ↓              ↓                   ↓
EngineSession  RecoveryUnit    Collection API
  ↓              ↓                   ↓
  └──────────────┴──────┬────────────┘
                        ↓
           ┌────────────┴────────────┐
           ↓                         ↓
      RecordStore              SortedDataInterface
    (聚簇记录存储)                 (索引接口)
           ↓                         ↓
           └────────────┬────────────┘
                        ↓
              Physical BTree Storage
                  (B+Tree实现)
                        ↓
              [buffer + pages]
```

## 核心组件

### 1. KVEngine (键值存储引擎)

**文件**: `server/storage/kv_engine.go`

**职责**:
- 管理底层存储资源
- 会话管理 (Session Management)
- RecordStore 和 Index 的生命周期管理
- 资源统计和监控

**核心接口**:
```go
type KVEngine interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    
    CreateSession(ctx context.Context) (EngineSession, error)
    
    GetRecordStore(namespace string) (RecordStore, error)
    CreateRecordStore(namespace string) (RecordStore, error)
    DropRecordStore(namespace string) error
    
    GetSortedDataInterface(namespace, indexName string) (SortedDataInterface, error)
    CreateSortedDataInterface(namespace, indexName string, unique bool) (SortedDataInterface, error)
    DropSortedDataInterface(namespace, indexName string) error
    
    GetStats() map[string]interface{}
}
```

**实现**: `WiredTigerKVEngine`
- 内部维护 RecordStore 和 Index 的映射
- 使用 namespace (database.collection) 作为标识
- 支持最大会话数限制
- 提供缓存大小配置

### 2. RecordStore (记录存储)

**文件**: `server/storage/record_store.go`

**职责**:
- 存储完整的 BSON 文档
- 使用 RecordId 作为主键（聚簇索引）
- 支持文档的 CRUD 操作
- 提供游标扫描能力

**核心接口**:
```go
type RecordStore interface {
    InsertRecord(ctx context.Context, recordId RecordId, data []byte) error
    UpdateRecord(ctx context.Context, recordId RecordId, data []byte) error
    DeleteRecord(ctx context.Context, recordId RecordId) error
    GetRecord(ctx context.Context, recordId RecordId) ([]byte, error)
    
    Scan(ctx context.Context, startId RecordId) (RecordCursor, error)
    
    NumRecords() int64
    DataSize() int64
    Truncate(ctx context.Context) error
}
```

**实现**: `BTreeRecordStore`
- 基于 B+Tree 存储 `RecordId → BSON Document`
- RecordId 支持 int64 和 byte[] 两种形式
- 自动维护记录数和数据大小统计
- 提供高效的范围扫描

**数据格式**:
```
Key:   RecordId (8字节 int64 或 可变长度 byte[])
Value: BSON 文档二进制 (完整文档)
```

### 3. SortedDataInterface (索引接口)

**文件**: `server/storage/sorted_data.go`

**职责**:
- 管理有序索引数据
- 支持唯一索引和非唯一索引
- 提供精确查找和范围查询
- 维护索引键到 RecordId 的映射

**核心接口**:
```go
type SortedDataInterface interface {
    Insert(ctx context.Context, key []byte, recordId RecordId) error
    Remove(ctx context.Context, key []byte, recordId RecordId) error
    
    Seek(ctx context.Context, key []byte) (IndexCursor, error)
    SeekRange(ctx context.Context, startKey, endKey []byte) (IndexCursor, error)
    
    NumEntries() int64
    IsEmpty() bool
    Clear(ctx context.Context) error
}
```

**实现**: `BTreeIndex`
- 基于 B+Tree 存储 `(IndexKey + RecordId) → RecordId`
- 组合键设计确保非唯一索引的多值支持
- 唯一索引在插入时检查键是否存在
- 提供游标迭代访问

**数据格式**:
```
Composite Key: [keyLen(4字节)][indexKey][recordId]
Value:         recordId (冗余存储便于查询)
```

### 4. RecoveryUnit (事务和快照抽象层)

**文件**: `server/storage/recovery_unit.go`

**职责**:
- 事务生命周期管理
- 快照隔离（Snapshot Isolation）
- 变更跟踪和回滚
- 时间戳管理（MVCC预留）

**核心接口**:
```go
type RecoveryUnit interface {
    BeginTransaction(ctx context.Context) error
    Commit(ctx context.Context) error
    Rollback(ctx context.Context) error
    
    GetReadTimestamp() time.Time
    SetCommitTimestamp(ts time.Time) error
    
    PrepareForHistoryStore(oldValue []byte) error  // MVCC 预留接口
    
    IsActive() bool
    IsCommitted() bool
    IsAborted() bool
    
    RegisterChange(change Change) error
}
```

**实现**: `WiredTigerRecoveryUnit`
- 维护事务状态机 (Inactive → Active → Committed/Aborted)
- 使用 Change 接口实现通用的变更跟踪
- 支持事务提交和回滚
- 预留 MVCC 历史存储接口

**事务状态流转**:
```
TxnStateInactive
      ↓ BeginTransaction()
TxnStateActive
      ↓ Commit() / Rollback()
TxnStateCommitted / TxnStateAborted
```

### 5. EngineSession (会话管理)

**文件**: `server/storage/session.go`

**职责**:
- 维护会话状态和上下文
- 关联 RecoveryUnit（事务）
- 会话级别的事务操作
- 资源生命周期管理

**核心接口**:
```go
type EngineSession interface {
    Begin(ctx context.Context) error
    End(ctx context.Context) error
    
    GetRecoveryUnit() RecoveryUnit
    GetSessionId() string
    
    BeginTransaction(ctx context.Context) error
    CommitTransaction(ctx context.Context) error
    RollbackTransaction(ctx context.Context) error
    
    IsActive() bool
    InTransaction() bool
}
```

**实现**: `WiredTigerSession`
- 每个会话持有一个 RecoveryUnit
- 会话结束时自动回滚未提交的事务
- 使用 UUID 作为会话标识
- 跟踪会话活动状态

### 6. BTree (物理存储层)

**文件**: `server/storage/btree/btree.go`

**职责**:
- B+Tree 数据结构实现
- 有序键值对存储
- 节点分裂和平衡
- 范围查询支持

**核心特性**:
- 可配置的阶数 (默认128)
- 叶子节点链表（支持高效范围扫描）
- 自动节点分裂和树平衡
- 线程安全 (RWMutex)

**数据结构**:
```go
type Node struct {
    isLeaf   bool
    keys     [][]byte  // 键列表
    values   [][]byte  // 值列表（仅叶子节点）
    children []*Node   // 子节点列表（仅内部节点）
    next     *Node     // 下一个叶子节点（叶子节点链表）
    parent   *Node     // 父节点
}
```

### 7. RecordId (记录标识符)

**文件**: `server/storage/record_id.go`

**职责**:
- 唯一标识一条记录
- 支持 int64 和 byte[] 两种表示
- 提供比较和序列化能力

**类型设计**:
```go
type RecordId struct {
    repr int8      // 0=null, 1=int64, 2=bytes
    long int64     // int64 类型的 RecordId
    data []byte    // byte[] 类型的 RecordId
}
```

**使用场景**:
- **int64**: 自增 ID，适合大部分场景，占用空间小
- **byte[]**: 支持复杂的分布式 ID、UUID 等

## 数据流示例

### 插入文档流程

```
1. Client → Protocol Layer
   MongoDB 客户端发送 OP_INSERT 消息
   
2. Protocol Layer → Engine API
   协议层解析消息，调用 Engine.Insert()
   
3. Engine API → KVEngine
   WiredTigerEngine 获取对应的 Collection
   
4. Collection → RecordStore
   生成 RecordId (自增)
   将 BSON 文档序列化
   调用 RecordStore.InsertRecord(recordId, bsonData)
   
5. RecordStore → BTree
   BTreeRecordStore 调用 btree.Insert(recordId_bytes, bsonData)
   
6. BTree 存储
   找到叶子节点位置
   插入键值对
   必要时分裂节点
   
7. Collection → Index
   遍历所有索引
   提取索引键
   调用 SortedDataInterface.Insert(indexKey, recordId)
   
8. Index → BTree
   BTreeIndex 调用 btree.Insert(compositeKey, recordId)
   组合键 = indexKey + recordId (确保唯一性)
   
9. 返回成功
   层层返回，最终响应客户端
```

### 查询文档流程（使用索引）

```
1. Client → Protocol Layer
   MongoDB 客户端发送 OP_QUERY 消息
   
2. Protocol Layer → Engine API
   协议层解析查询条件，调用 Engine.Find()
   
3. Engine API → Collection
   WiredTigerEngine 获取对应的 Collection
   
4. Collection → Index
   如果查询条件匹配索引，使用索引查找
   调用 SortedDataInterface.Seek(indexKey)
   
5. Index → BTree
   BTreeIndex 执行范围查询
   返回 IndexCursor
   
6. IndexCursor → RecordId
   遍历游标，获取每个 RecordId
   
7. RecordStore → 获取文档
   使用 RecordId 从 RecordStore 读取完整文档
   调用 RecordStore.GetRecord(recordId)
   
8. RecordStore → BTree
   BTreeRecordStore 查找 recordId → bsonData
   
9. 返回结果
   将 BSON 数据反序列化为 Document
   应用过滤条件
   返回给客户端
```

### 事务提交流程

```
1. Client 开始事务
   
2. Session.BeginTransaction()
   创建 EngineSession
   调用 RecoveryUnit.BeginTransaction()
   
3. 执行多个操作
   每个操作注册 Change 到 RecoveryUnit
   RecoveryUnit.RegisterChange(change)
   
4. Client 提交事务
   
5. Session.CommitTransaction()
   调用 RecoveryUnit.Commit()
   
6. RecoveryUnit 提交所有变更
   遍历所有 Change
   调用 change.Commit()
   
7. 持久化数据
   所有变更应用到 BTree
   更新统计信息
   
8. 标记事务已提交
   state = TxnStateCommitted
   
9. 返回成功
```

## 性能优化点

### 已实现
1. **B+Tree 叶子节点链表**: 支持高效的范围扫描
2. **读写锁**: RecordStore 和 Index 使用 RWMutex 提高并发
3. **原子计数器**: 统计信息使用 atomic 操作避免锁竞争
4. **组合键设计**: 索引支持非唯一键，避免额外的数据结构
5. **节点预分配**: B+Tree 使用 slice 预分配减少内存分配

### 待优化（未来）
1. **缓冲池管理**: 实现页面缓存池，减少内存分配
2. **压缩存储**: BSON 文档压缩存储
3. **异步刷盘**: 批量写入和异步持久化
4. **MVCC 完整实现**: 多版本并发控制和历史存储
5. **检查点机制**: 定期将内存数据持久化到磁盘
6. **预写日志 (WAL)**: 事务日志和崩溃恢复

## 测试覆盖

所有核心组件均有单元测试覆盖：

- `TestKVEngine`: KV 引擎完整测试
- `TestRecoveryUnit`: 事务提交和回滚测试
- `TestBTreeRecordStore`: RecordStore CRUD 和扫描测试
- `TestSortedDataInterface`: 索引插入、查找、范围查询测试

性能基准测试：

- `BenchmarkRecordStoreInsert`: 记录插入性能
- `BenchmarkRecordStoreGet`: 记录读取性能
- `BenchmarkIndexInsert`: 索引插入性能

运行测试：
```bash
# 运行所有测试
go test -v ./server/storage/

# 运行性能测试
go test -bench=. ./server/storage/

# 运行特定测试
go test -v ./server/storage/ -run TestKVEngine
```

## 配置参数

### KVEngine 配置
```go
type KVEngineConfig struct {
    CacheSize         int64  // 缓存大小（字节），默认 1GB
    MaxSessions       int    // 最大会话数，默认 1000
    CheckpointEnabled bool   // 是否启用检查点
}
```

### BTree 配置
```go
// B+Tree 阶数（每个节点最多的子节点数）
// 默认 128，可根据场景调整
btree.NewBTree(order int)
```

## 文件组织

```
server/storage/
├── btree/
│   └── btree.go           # B+Tree 物理存储实现
├── engine.go              # Engine 高层接口和实现
├── kv_engine.go           # KVEngine 核心引擎
├── record_store.go        # RecordStore 记录存储
├── sorted_data.go         # SortedDataInterface 索引接口
├── recovery_unit.go       # RecoveryUnit 事务抽象
├── session.go             # EngineSession 会话管理
├── record_id.go           # RecordId 类型定义
└── storage_test.go        # 完整测试套件
```

## 与 MongoDB 的对应关系

| MongoDB 组件 | XMongoDB 组件 | 说明 |
|-------------|--------------|------|
| WiredTiger Engine | WiredTigerKVEngine | 底层存储引擎 |
| Collection | RecordStore | 文档存储 |
| Index | SortedDataInterface | 索引管理 |
| Session | EngineSession | 会话和事务上下文 |
| RecoveryUnit | RecoveryUnit | 事务和快照 |
| RecordId | RecordId | 记录标识符 |
| B-Tree | BTree | 物理存储结构 |

## 扩展性设计

### 接口化设计
所有核心组件都定义了接口，方便替换实现：
- 可以替换 BTree 为其他数据结构（LSM-Tree、SkipList 等）
- 可以实现不同的 RecoveryUnit（支持不同的事务模型）
- 可以扩展 RecordId 支持更多类型

### 插件化存储引擎
通过 `NewEngine(config)` 工厂函数，可以轻松支持多种存储引擎：
```go
switch cfg.Engine {
case "wiredTiger":
    return NewWiredTigerEngine(cfg)
case "memory":
    return NewMemoryEngine(cfg)
case "rocksdb":
    return NewRocksDBEngine(cfg)  // 未来扩展
}
```

## 总结

XMongoDB 存储引擎实现了完整的分层架构，从协议层到物理存储层层清晰，职责明确。核心特性包括：

✅ **完整的事务支持**: RecoveryUnit 提供事务抽象  
✅ **高效的索引**: 基于 B+Tree 的 SortedDataInterface  
✅ **灵活的 RecordId**: 支持 int64 和 byte[] 两种形式  
✅ **可扩展架构**: 接口化设计便于替换实现  
✅ **MVCC 预留**: PrepareForHistoryStore 接口为多版本控制预留  
✅ **完整测试覆盖**: 单元测试和性能基准测试  

数据流清晰：**Client → Protocol → Engine → KVEngine → RecordStore/Index → BTree**
