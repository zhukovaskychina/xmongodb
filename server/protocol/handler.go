package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/AlexStocks/getty"
)

// PackageHandler MongoDB 协议包处理器
type PackageHandler struct{}

// NewPackageHandler 创建新的包处理器
func NewPackageHandler() *PackageHandler {
	return &PackageHandler{}
}

// Read 读取数据包
func (h *PackageHandler) Read(ss getty.Session, data []byte) (interface{}, int, error) {
	if len(data) < 16 {
		// MongoDB 消息头至少16字节
		return nil, 0, nil
	}

	// 解析 MongoDB 消息头
	header, err := parseMessageHeader(data)
	if err != nil {
		return nil, 0, fmt.Errorf("解析消息头失败: %w", err)
	}

	// 检查是否有完整的消息
	if len(data) < int(header.MessageLength) {
		return nil, 0, nil // 等待更多数据
	}

	// 解析完整消息
	message, err := parseMessage(data[:header.MessageLength], header)
	if err != nil {
		return nil, 0, fmt.Errorf("解析消息失败: %w", err)
	}

	return message, int(header.MessageLength), nil
}

// Write 写入数据包
func (h *PackageHandler) Write(ss getty.Session, pkg interface{}) ([]byte, error) {
	message, ok := pkg.(*Message)
	if !ok {
		return nil, fmt.Errorf("无效的消息类型")
	}

	return message.Serialize()
}

// MessageHeader MongoDB 消息头
type MessageHeader struct {
	MessageLength int32 // 消息总长度
	RequestID     int32 // 请求ID
	ResponseTo    int32 // 响应的请求ID
	OpCode        int32 // 操作码
}

// Message MongoDB 消息
type Message struct {
	Header *MessageHeader
	Body   []byte
	OpCode OpCode
}

// OpCode 操作码
type OpCode int32

const (
	OpReply        OpCode = 1    // 回复消息
	OpUpdate       OpCode = 2001 // 更新文档
	OpInsert       OpCode = 2002 // 插入文档
	OpQuery        OpCode = 2004 // 查询文档
	OpGetMore      OpCode = 2005 // 获取更多结果
	OpDelete       OpCode = 2006 // 删除文档
	OpKillCursors  OpCode = 2007 // 关闭游标
	OpCommand      OpCode = 2010 // 命令 (MongoDB 3.2+)
	OpCommandReply OpCode = 2011 // 命令回复
	OpMsg          OpCode = 2013 // 消息 (MongoDB 3.6+)
)

// parseMessageHeader 解析消息头
func parseMessageHeader(data []byte) (*MessageHeader, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("数据长度不足")
	}

	reader := bytes.NewReader(data)
	header := &MessageHeader{}

	if err := binary.Read(reader, binary.LittleEndian, &header.MessageLength); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &header.RequestID); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &header.ResponseTo); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &header.OpCode); err != nil {
		return nil, err
	}

	return header, nil
}

// parseMessage 解析完整消息
func parseMessage(data []byte, header *MessageHeader) (*Message, error) {
	message := &Message{
		Header: header,
		Body:   data[16:], // 跳过16字节头部
		OpCode: OpCode(header.OpCode),
	}

	return message, nil
}

// Serialize 序列化消息
func (m *Message) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// 写入消息头
	if err := binary.Write(buf, binary.LittleEndian, m.Header.MessageLength); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, m.Header.RequestID); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, m.Header.ResponseTo); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, m.Header.OpCode); err != nil {
		return nil, err
	}

	// 写入消息体
	if _, err := buf.Write(m.Body); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// GetOpCodeName 获取操作码名称
func (op OpCode) String() string {
	switch op {
	case OpReply:
		return "OP_REPLY"
	case OpUpdate:
		return "OP_UPDATE"
	case OpInsert:
		return "OP_INSERT"
	case OpQuery:
		return "OP_QUERY"
	case OpGetMore:
		return "OP_GET_MORE"
	case OpDelete:
		return "OP_DELETE"
	case OpKillCursors:
		return "OP_KILL_CURSORS"
	case OpCommand:
		return "OP_COMMAND"
	case OpCommandReply:
		return "OP_COMMAND_REPLY"
	case OpMsg:
		return "OP_MSG"
	default:
		return "UNKNOWN"
	}
}
