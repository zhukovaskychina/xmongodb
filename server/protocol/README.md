# MongoDB Wire Protocol 协议层

本目录包含从 `mongo-go-driver-release-2.4` 提取的 MongoDB Wire Protocol 核心协议实现。

## 📁 目录结构

```
protocol/
├── wiremessage/     # Wire Protocol 消息处理
│   └── wiremessage.go
└── bsoncore/        # BSON 编解码核心
    ├── bsoncore.go
    ├── document.go
    ├── element.go
    ├── value.go
    ├── array.go
    ├── type.go
    ├── decimal128.go
    ├── util.go
    └── ...
```

## 🔧 主要组件

### 1. Wire Message 包 (wiremessage)

实现了 MongoDB Wire Protocol 的消息格式处理：

#### OpCode（操作码）
- `OpReply` (1) - 服务器回复
- `OpMsg` (2013) - 现代 MongoDB 消息格式（推荐使用）
- `OpQuery` (2004) - 查询操作（已弃用）
- `OpInsert` (2002) - 插入操作
- `OpUpdate` (2001) - 更新操作
- `OpDelete` (2006) - 删除操作
- `OpGetMore` (2005) - 获取更多数据
- `OpKillCursors` (2007) - 关闭游标
- `OpCompressed` (2012) - 压缩消息

#### 消息标志
- **QueryFlag**: 查询标志（TailableCursor, SecondaryOK, AwaitData 等）
- **MsgFlag**: OP_MSG 标志（ChecksumPresent, MoreToCome, ExhaustAllowed）
- **ReplyFlag**: 回复标志（CursorNotFound, QueryFailure 等）

#### 压缩支持
- CompressorNoOp - 无压缩
- CompressorSnappy - Snappy 压缩
- CompressorZLib - ZLib 压缩
- CompressorZstd - Zstd 压缩

### 2. BSON Core 包 (bsoncore)

实现了 BSON (Binary JSON) 的编解码：

#### 核心类型
- **Document** - BSON 文档
- **Array** - BSON 数组
- **Element** - BSON 元素
- **Value** - BSON 值
- **Type** - BSON 类型枚举

#### BSON 数据类型
- TypeDouble (1) - 64位浮点数
- TypeString (2) - UTF-8 字符串
- TypeEmbeddedDocument (3) - 嵌入文档
- TypeArray (4) - 数组
- TypeBinary (5) - 二进制数据
- TypeObjectID (7) - ObjectID
- TypeBoolean (8) - 布尔值
- TypeDateTime (9) - UTC 日期时间
- TypeNull (10) - Null
- TypeRegex (11) - 正则表达式
- TypeInt32 (16) - 32位整数
- TypeTimestamp (17) - 时间戳
- TypeInt64 (18) - 64位整数
- TypeDecimal128 (19) - 128位十进制

## 📝 使用示例

### 创建 Wire Message Header

```go
import "github.com/zhukovaskychina/xmongodb/server/protocol/wiremessage"

// 创建消息头
requestID := wiremessage.NextRequestID()
index, message := wiremessage.AppendHeaderStart(nil, requestID, 0, wiremessage.OpMsg)

// 读取消息头
length, reqID, respTo, opcode, remainder, ok := wiremessage.ReadHeader(wireMsg)
if ok {
    fmt.Printf("OpCode: %s, RequestID: %d\n", opcode, reqID)
}
```

### BSON 文档操作

```go
import "github.com/zhukovaskychina/xmongodb/server/protocol/bsoncore"

// 创建 BSON 文档
idx, doc := bsoncore.AppendDocumentStart(nil)
doc = bsoncore.AppendStringElement(doc, "name", "MongoDB")
doc = bsoncore.AppendInt32Element(doc, "port", 27017)
doc, _ = bsoncore.AppendDocumentEnd(doc, idx)

// 读取 BSON 文档
document := bsoncore.Document(doc)
nameValue := document.Lookup("name")
if str, ok := nameValue.StringValueOK(); ok {
    fmt.Printf("Name: %s\n", str)
}
```

### OP_MSG 消息格式

```go
// 添加 OP_MSG 标志
msg := wiremessage.AppendMsgFlags(nil, wiremessage.ExhaustAllowed)

// 添加 Section Type 0 (单文档)
msg = wiremessage.AppendMsgSectionType(msg, wiremessage.SingleDocument)

// 添加文档内容
idx, msg := bsoncore.AppendDocumentStart(msg)
msg = bsoncore.AppendStringElement(msg, "find", "users")
msg = bsoncore.AppendInt32Element(msg, "limit", 10)
msg, _ = bsoncore.AppendDocumentEnd(msg, idx)
```

## 🔍 关键特性

1. **零拷贝设计** - 直接操作字节切片，避免不必要的内存分配
2. **流式处理** - 支持 MoreToCome 和 ExhaustAllowed 标志的流式响应
3. **压缩支持** - 支持多种压缩算法（Snappy, ZLib, Zstd）
4. **类型安全** - 强类型的 BSON 值访问
5. **错误处理** - 完善的错误检测和验证

## 📚 相关规范

- [MongoDB Wire Protocol](https://www.mongodb.com/docs/manual/reference/mongodb-wire-protocol/)
- [BSON Specification](http://bsonspec.org/)
- [OP_MSG Specification](https://github.com/mongodb/specifications/blob/master/source/message/OP_MSG.rst)

## ⚠️ 注意事项

1. **大端/小端** - 所有整数使用小端字节序（Little Endian）
2. **字符串编码** - 所有字符串使用 UTF-8 编码
3. **CString** - 键名和某些字符串以 null 字节（0x00）结尾
4. **文档长度** - 文档的前4个字节表示整个文档的字节长度（包括这4个字节）
5. **消息长度** - Wire Message 的前4个字节表示整个消息的字节长度

## 🔄 版本兼容性

本实现基于 MongoDB Go Driver 2.4，支持：
- MongoDB 3.6+
- Wire Protocol Version 6+
- OP_MSG (首选)
- OP_QUERY (向后兼容)

## 📖 扩展阅读

如需深入了解协议细节，请参考：
- `wiremessage/wiremessage.go` - Wire Protocol 实现
- `bsoncore/bsoncore.go` - BSON 核心函数
- `bsoncore/document.go` - 文档操作
- `bsoncore/value.go` - 值类型处理
