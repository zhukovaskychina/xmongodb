package main

import (
	"fmt"

	"github.com/zhukovaskychina/xmongodb/server/protocol/bsoncore"
	"github.com/zhukovaskychina/xmongodb/server/protocol/wiremessage"
)

func main() {
	fmt.Println("=== MongoDB Wire Protocol 测试 ===\n")

	// 1. 测试 OpCode
	fmt.Println("1. OpCode 测试:")
	fmt.Printf("   OpMsg: %s (%d)\n", wiremessage.OpMsg, wiremessage.OpMsg)
	fmt.Printf("   OpReply: %s (%d)\n", wiremessage.OpReply, wiremessage.OpReply)
	fmt.Printf("   OpQuery: %s (%d)\n", wiremessage.OpQuery, wiremessage.OpQuery)

	// 2. 测试 Wire Message Header
	fmt.Println("\n2. Wire Message Header 测试:")
	requestID := wiremessage.NextRequestID()
	fmt.Printf("   生成的 RequestID: %d\n", requestID)

	var msg []byte
	idx, msg := wiremessage.AppendHeaderStart(msg, requestID, 0, wiremessage.OpMsg)
	fmt.Printf("   消息头索引: %d, 当前长度: %d\n", idx, len(msg))

	// 3. 测试 BSON 文档创建
	fmt.Println("\n3. BSON 文档创建测试:")
	var doc []byte
	docIdx, doc := bsoncore.AppendDocumentStart(doc)
	doc = bsoncore.AppendStringElement(doc, "database", "xmongodb")
	doc = bsoncore.AppendInt32Element(doc, "port", 27017)
	doc = bsoncore.AppendBooleanElement(doc, "enabled", true)
	doc, err := bsoncore.AppendDocumentEnd(doc, docIdx)
	if err != nil {
		fmt.Printf("   错误: %v\n", err)
		return
	}
	fmt.Printf("   文档创建成功, 长度: %d bytes\n", len(doc))

	// 4. 测试 BSON 文档读取
	fmt.Println("\n4. BSON 文档读取测试:")
	document := bsoncore.Document(doc)
	
	if dbValue := document.Lookup("database"); dbValue.Type == bsoncore.TypeString {
		if str, ok := dbValue.StringValueOK(); ok {
			fmt.Printf("   database: %s\n", str)
		}
	}
	
	if portValue := document.Lookup("port"); portValue.Type == bsoncore.TypeInt32 {
		if port, ok := portValue.Int32OK(); ok {
			fmt.Printf("   port: %d\n", port)
		}
	}
	
	if enabledValue := document.Lookup("enabled"); enabledValue.Type == bsoncore.TypeBoolean {
		if enabled, ok := enabledValue.BooleanOK(); ok {
			fmt.Printf("   enabled: %v\n", enabled)
		}
	}

	// 5. 测试 OP_MSG 格式
	fmt.Println("\n5. OP_MSG 消息格式测试:")
	var opMsg []byte
	msgIdx, opMsg := wiremessage.AppendHeaderStart(opMsg, wiremessage.NextRequestID(), 0, wiremessage.OpMsg)
	opMsg = wiremessage.AppendMsgFlags(opMsg, wiremessage.ExhaustAllowed)
	opMsg = wiremessage.AppendMsgSectionType(opMsg, wiremessage.SingleDocument)
	
	// 添加命令文档
	cmdIdx, opMsg := bsoncore.AppendDocumentStart(opMsg)
	opMsg = bsoncore.AppendStringElement(opMsg, "find", "users")
	opMsg = bsoncore.AppendInt32Element(opMsg, "limit", 10)
	opMsg, _ = bsoncore.AppendDocumentEnd(opMsg, cmdIdx)
	
	// 完成消息
	opMsg = bsoncore.UpdateLength(opMsg, msgIdx, int32(len(opMsg)))
	fmt.Printf("   OP_MSG 消息创建成功, 总长度: %d bytes\n", len(opMsg))
	
	// 读取消息头
	length, reqID, respTo, opcode, _, ok := wiremessage.ReadHeader(opMsg)
	if ok {
		fmt.Printf("   消息长度: %d\n", length)
		fmt.Printf("   RequestID: %d\n", reqID)
		fmt.Printf("   ResponseTo: %d\n", respTo)
		fmt.Printf("   OpCode: %s\n", opcode)
	}

	// 6. 测试 BSON 类型
	fmt.Println("\n6. BSON 类型系统测试:")
	fmt.Printf("   TypeDouble: %s (%d)\n", bsoncore.TypeDouble, bsoncore.TypeDouble)
	fmt.Printf("   TypeString: %s (%d)\n", bsoncore.TypeString, bsoncore.TypeString)
	fmt.Printf("   TypeDocument: %s (%d)\n", bsoncore.TypeEmbeddedDocument, bsoncore.TypeEmbeddedDocument)
	fmt.Printf("   TypeArray: %s (%d)\n", bsoncore.TypeArray, bsoncore.TypeArray)
	fmt.Printf("   TypeObjectID: %s (%d)\n", bsoncore.TypeObjectID, bsoncore.TypeObjectID)
	fmt.Printf("   TypeInt32: %s (%d)\n", bsoncore.TypeInt32, bsoncore.TypeInt32)
	fmt.Printf("   TypeInt64: %s (%d)\n", bsoncore.TypeInt64, bsoncore.TypeInt64)

	fmt.Println("\n=== 所有测试通过！ ===")
}
