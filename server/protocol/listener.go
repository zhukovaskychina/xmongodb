package protocol

import (
	"context"

	getty "github.com/apache/dubbo-getty"
	"github.com/zhukovaskychina/xmongodb/logger"
	"github.com/zhukovaskychina/xmongodb/server/storage"
)

// EventListener MongoDB 协议事件监听器
type EventListener struct {
	storageEngine storage.Engine
}

// NewEventListener 创建新的事件监听器
func NewEventListener(engine storage.Engine) *EventListener {
	return &EventListener{
		storageEngine: engine,
	}
}

// OnOpen 连接打开事件
func (l *EventListener) OnOpen(session getty.Session) error {
	logger.Infof("客户端连接: %s", session.RemoteAddr())
	return nil
}

// OnClose 连接关闭事件
func (l *EventListener) OnClose(session getty.Session) {
	logger.Infof("客户端断开: %s", session.RemoteAddr())
}

// OnMessage 消息接收事件
func (l *EventListener) OnMessage(session getty.Session, pkg interface{}) {
	message, ok := pkg.(*Message)
	if !ok {
		logger.Errorf("收到无效消息类型")
		return
	}

	logger.Debugf("收到消息: OpCode=%s, RequestID=%d", message.OpCode, message.Header.RequestID)

	// 处理消息
	response := l.handleMessage(session, message)
	if response != nil {
		if err := session.WritePkg(response); err != nil {
			logger.Errorf("发送响应失败: %v", err)
		}
	}
}

// OnError 错误事件
func (l *EventListener) OnError(session getty.Session, err error) {
	logger.Errorf("会话错误 %s: %v", session.RemoteAddr(), err)
}

// OnCron 定时事件
func (l *EventListener) OnCron(session getty.Session) {
	// 可以在这里实现心跳检测等定时任务
}

// handleMessage 处理具体的消息
func (l *EventListener) handleMessage(session getty.Session, message *Message) *Message {
	ctx := context.Background()

	switch message.OpCode {
	case OpQuery:
		return l.handleQuery(ctx, message)
	case OpInsert:
		return l.handleInsert(ctx, message)
	case OpUpdate:
		return l.handleUpdate(ctx, message)
	case OpDelete:
		return l.handleDelete(ctx, message)
	case OpCommand:
		return l.handleCommand(ctx, message)
	case OpMsg:
		return l.handleMsg(ctx, message)
	default:
		logger.Warnf("不支持的操作码: %s", message.OpCode)
		return l.createErrorResponse(message, "不支持的操作")
	}
}

// handleQuery 处理查询操作
func (l *EventListener) handleQuery(ctx context.Context, message *Message) *Message {
	// TODO: 实现查询逻辑
	logger.Debug("处理查询操作")
	return l.createSuccessResponse(message, []byte("查询结果"))
}

// handleInsert 处理插入操作
func (l *EventListener) handleInsert(ctx context.Context, message *Message) *Message {
	// TODO: 实现插入逻辑
	logger.Debug("处理插入操作")
	return l.createSuccessResponse(message, []byte("插入成功"))
}

// handleUpdate 处理更新操作
func (l *EventListener) handleUpdate(ctx context.Context, message *Message) *Message {
	// TODO: 实现更新逻辑
	logger.Debug("处理更新操作")
	return l.createSuccessResponse(message, []byte("更新成功"))
}

// handleDelete 处理删除操作
func (l *EventListener) handleDelete(ctx context.Context, message *Message) *Message {
	// TODO: 实现删除逻辑
	logger.Debug("处理删除操作")
	return l.createSuccessResponse(message, []byte("删除成功"))
}

// handleCommand 处理命令操作
func (l *EventListener) handleCommand(ctx context.Context, message *Message) *Message {
	// TODO: 实现命令逻辑
	logger.Debug("处理命令操作")
	return l.createSuccessResponse(message, []byte("命令执行成功"))
}

// handleMsg 处理消息操作 (MongoDB 3.6+)
func (l *EventListener) handleMsg(ctx context.Context, message *Message) *Message {
	// TODO: 实现消息处理逻辑
	logger.Debug("处理消息操作")
	return l.createSuccessResponse(message, []byte("消息处理成功"))
}

// createSuccessResponse 创建成功响应
func (l *EventListener) createSuccessResponse(request *Message, data []byte) *Message {
	response := &Message{
		Header: &MessageHeader{
			MessageLength: int32(16 + len(data)),
			RequestID:     generateRequestID(),
			ResponseTo:    request.Header.RequestID,
			OpCode:        int32(OpReply),
		},
		Body:   data,
		OpCode: OpReply,
	}
	return response
}

// createErrorResponse 创建错误响应
func (l *EventListener) createErrorResponse(request *Message, errorMsg string) *Message {
	data := []byte(errorMsg)
	response := &Message{
		Header: &MessageHeader{
			MessageLength: int32(16 + len(data)),
			RequestID:     generateRequestID(),
			ResponseTo:    request.Header.RequestID,
			OpCode:        int32(OpReply),
		},
		Body:   data,
		OpCode: OpReply,
	}
	return response
}

// generateRequestID 生成请求ID
func generateRequestID() int32 {
	// TODO: 实现更好的ID生成策略
	return 1
}
