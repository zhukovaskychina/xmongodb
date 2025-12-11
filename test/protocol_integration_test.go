package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/zhukovaskychina/xmongodb/server/protocol/bsoncore"
	"github.com/zhukovaskychina/xmongodb/server/protocol/wiremessage"
)

func main() {
	fmt.Println("=== MongoDB 协议层与 Getty 集成测试 ===\n")

	// 等待服务器启动
	time.Sleep(2 * time.Second)

	// 连接到服务器
	conn, err := net.Dial("tcp", "127.0.0.1:27017")
	if err != nil {
		fmt.Printf("连接失败: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Println("✓ 成功连接到 127.0.0.1:27017")

	// 1. 测试发送 OP_MSG 消息
	fmt.Println("\n1. 发送 OP_MSG 消息测试:")
	if err := testOpMsg(conn); err != nil {
		fmt.Printf("   ✗ OP_MSG 测试失败: %v\n", err)
	} else {
		fmt.Println("   ✓ OP_MSG 测试成功")
	}

	// 2. 测试发送 OP_QUERY 消息
	fmt.Println("\n2. 发送 OP_QUERY 消息测试:")
	if err := testOpQuery(conn); err != nil {
		fmt.Printf("   ✗ OP_QUERY 测试失败: %v\n", err)
	} else {
		fmt.Println("   ✓ OP_QUERY 测试成功")
	}

	fmt.Println("\n=== 测试完成 ===")
}

// testOpMsg 测试 OP_MSG 协议
func testOpMsg(conn net.Conn) error {
	// 构造 OP_MSG 消息
	var msg []byte
	
	// 消息头
	idx, msg := wiremessage.AppendHeaderStart(msg, wiremessage.NextRequestID(), 0, wiremessage.OpMsg)
	
	// OP_MSG 标志
	msg = wiremessage.AppendMsgFlags(msg, 0)
	
	// Section Type 0 (单文档)
	msg = wiremessage.AppendMsgSectionType(msg, wiremessage.SingleDocument)
	
	// 构造命令文档
	docIdx, msg := bsoncore.AppendDocumentStart(msg)
	msg = bsoncore.AppendStringElement(msg, "hello", "xmongodb")
	msg = bsoncore.AppendInt32Element(msg, "version", 1)
	msg, _ = bsoncore.AppendDocumentEnd(msg, docIdx)
	
	// 更新消息长度
	msg = bsoncore.UpdateLength(msg, idx, int32(len(msg)))
	
	// 发送消息
	if _, err := conn.Write(msg); err != nil {
		return fmt.Errorf("发送失败: %w", err)
	}
	
	// 接收响应
	response := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err := conn.Read(response)
	if err != nil {
		return fmt.Errorf("接收响应失败: %w", err)
	}
	
	if n > 0 {
		fmt.Printf("   接收到 %d 字节响应\n", n)
		// 解析响应头
		length, reqID, respTo, opcode, _, ok := wiremessage.ReadHeader(response[:n])
		if ok {
			fmt.Printf("   响应: Length=%d, RequestID=%d, ResponseTo=%d, OpCode=%d\n", 
				length, reqID, respTo, opcode)
		}
	}
	
	return nil
}

// testOpQuery 测试 OP_QUERY 协议
func testOpQuery(conn net.Conn) error {
	// 构造简单的 OP_QUERY 消息
	buf := new(bytes.Buffer)
	
	// 消息头（稍后更新长度）
	binary.Write(buf, binary.LittleEndian, int32(0)) // 消息长度（占位）
	binary.Write(buf, binary.LittleEndian, int32(2)) // RequestID
	binary.Write(buf, binary.LittleEndian, int32(0)) // ResponseTo
	binary.Write(buf, binary.LittleEndian, int32(2004)) // OpCode (OP_QUERY)
	
	// OP_QUERY 字段
	binary.Write(buf, binary.LittleEndian, int32(0))     // flags
	buf.WriteString("test.collection\x00")               // 集合名
	binary.Write(buf, binary.LittleEndian, int32(0))     // numberToSkip
	binary.Write(buf, binary.LittleEndian, int32(1))     // numberToReturn
	
	// 查询文档
	var doc []byte
	docIdx, doc := bsoncore.AppendDocumentStart(doc)
	doc = bsoncore.AppendStringElement(doc, "find", "test")
	doc, _ = bsoncore.AppendDocumentEnd(doc, docIdx)
	buf.Write(doc)
	
	// 更新消息长度
	msg := buf.Bytes()
	binary.LittleEndian.PutUint32(msg[0:4], uint32(len(msg)))
	
	// 发送消息
	if _, err := conn.Write(msg); err != nil {
		return fmt.Errorf("发送失败: %w", err)
	}
	
	// 接收响应
	response := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err := conn.Read(response)
	if err != nil {
		return fmt.Errorf("接收响应失败: %w", err)
	}
	
	if n > 0 {
		fmt.Printf("   接收到 %d 字节响应\n", n)
		// 解析响应头
		length, reqID, respTo, opcode, _, ok := wiremessage.ReadHeader(response[:n])
		if ok {
			fmt.Printf("   响应: Length=%d, RequestID=%d, ResponseTo=%d, OpCode=%d\n", 
				length, reqID, respTo, opcode)
		}
	}
	
	return nil
}
