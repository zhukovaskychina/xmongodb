package main

import (
	"context"
	"fmt"
	"log"
	
	"github.com/zhukovaskychina/xmongodb/server/storage"
)

// 演示完整的存储引擎使用流程
func main() {
	ctx := context.Background()
	
	// 1. 创建 KV 引擎
	fmt.Println("=== 步骤 1: 创建 KV 引擎 ===")
	config := storage.KVEngineConfig{
		CacheSize:         100 * 1024 * 1024, // 100MB
		MaxSessions:       100,
		CheckpointEnabled: true,
	}
	kvEngine := storage.NewKVEngine(config)
	
	// 启动引擎
	if err := kvEngine.Start(ctx); err != nil {
		log.Fatalf("启动引擎失败: %v", err)
	}
	defer kvEngine.Stop(ctx)
	fmt.Println("✓ KV 引擎已启动")
	
	// 2. 创建会话
	fmt.Println("\n=== 步骤 2: 创建会话 ===")
	session, err := kvEngine.CreateSession(ctx)
	if err != nil {
		log.Fatalf("创建会话失败: %v", err)
	}
	defer session.End(ctx)
	fmt.Printf("✓ 会话已创建: %s\n", session.GetSessionId())
	
	// 3. 创建 RecordStore（集合）
	fmt.Println("\n=== 步骤 3: 创建 RecordStore ===")
	namespace := "testdb.users"
	recordStore, err := kvEngine.CreateRecordStore(namespace)
	if err != nil {
		log.Fatalf("创建 RecordStore 失败: %v", err)
	}
	fmt.Printf("✓ RecordStore 已创建: %s\n", namespace)
	
	// 4. 创建索引
	fmt.Println("\n=== 步骤 4: 创建索引 ===")
	idIndex, err := kvEngine.CreateSortedDataInterface(namespace, "_id_", true)
	if err != nil {
		log.Fatalf("创建 _id 索引失败: %v", err)
	}
	fmt.Println("✓ _id 唯一索引已创建")
	
	nameIndex, err := kvEngine.CreateSortedDataInterface(namespace, "name_idx", false)
	if err != nil {
		log.Fatalf("创建 name 索引失败: %v", err)
	}
	fmt.Println("✓ name 索引已创建")
	
	// 5. 开始事务
	fmt.Println("\n=== 步骤 5: 开始事务 ===")
	if err := session.BeginTransaction(ctx); err != nil {
		log.Fatalf("开始事务失败: %v", err)
	}
	fmt.Println("✓ 事务已开始")
	
	// 6. 插入文档
	fmt.Println("\n=== 步骤 6: 插入文档 ===")
	users := []struct {
		id   int64
		name string
		age  int
	}{
		{1, "Alice", 30},
		{2, "Bob", 25},
		{3, "Charlie", 35},
		{4, "Diana", 28},
		{5, "Eve", 32},
	}
	
	for _, user := range users {
		// 创建 RecordId
		recordId := storage.NewRecordIdFromLong(user.id)
		
		// 简化的 BSON 文档（实际应使用 bsoncore）
		doc := fmt.Sprintf(`{"_id":%d,"name":"%s","age":%d}`, user.id, user.name, user.age)
		data := []byte(doc)
		
		// 插入到 RecordStore
		if err := recordStore.InsertRecord(ctx, recordId, data); err != nil {
			log.Fatalf("插入记录失败: %v", err)
		}
		
		// 更新 _id 索引
		idKey := []byte(fmt.Sprintf("%d", user.id))
		if err := idIndex.Insert(ctx, idKey, recordId); err != nil {
			log.Fatalf("更新 _id 索引失败: %v", err)
		}
		
		// 更新 name 索引
		nameKey := []byte(user.name)
		if err := nameIndex.Insert(ctx, nameKey, recordId); err != nil {
			log.Fatalf("更新 name 索引失败: %v", err)
		}
		
		fmt.Printf("✓ 插入文档: id=%d, name=%s, age=%d\n", user.id, user.name, user.age)
	}
	
	// 7. 提交事务
	fmt.Println("\n=== 步骤 7: 提交事务 ===")
	if err := session.CommitTransaction(ctx); err != nil {
		log.Fatalf("提交事务失败: %v", err)
	}
	fmt.Println("✓ 事务已提交")
	
	// 8. 通过 RecordId 直接查询
	fmt.Println("\n=== 步骤 8: 通过 RecordId 查询 ===")
	recordId := storage.NewRecordIdFromLong(3)
	data, err := recordStore.GetRecord(ctx, recordId)
	if err != nil {
		log.Fatalf("查询记录失败: %v", err)
	}
	fmt.Printf("✓ 找到记录 (RecordId=3): %s\n", string(data))
	
	// 9. 通过索引查询
	fmt.Println("\n=== 步骤 9: 通过 name 索引查询 ===")
	nameKey := []byte("Alice")
	cursor, err := nameIndex.Seek(ctx, nameKey)
	if err != nil {
		log.Fatalf("索引查询失败: %v", err)
	}
	defer cursor.Close()
	
	if cursor.Next() {
		foundRecordId := cursor.RecordId()
		doc, _ := recordStore.GetRecord(ctx, foundRecordId)
		fmt.Printf("✓ 通过 name='Alice' 找到文档: %s\n", string(doc))
	}
	
	// 10. 范围扫描
	fmt.Println("\n=== 步骤 10: 范围扫描所有文档 ===")
	scanCursor, err := recordStore.Scan(ctx, storage.NullRecordId())
	if err != nil {
		log.Fatalf("扫描失败: %v", err)
	}
	defer scanCursor.Close()
	
	count := 0
	for scanCursor.Next() {
		count++
		recordId := scanCursor.RecordId()
		data := scanCursor.Data()
		fmt.Printf("  [%d] RecordId=%s, Data=%s\n", count, recordId.String(), string(data))
	}
	fmt.Printf("✓ 共扫描 %d 条记录\n", count)
	
	// 11. 更新文档
	fmt.Println("\n=== 步骤 11: 更新文档 ===")
	updateRecordId := storage.NewRecordIdFromLong(2)
	newData := []byte(`{"_id":2,"name":"Bob","age":26,"updated":true}`)
	
	if err := recordStore.UpdateRecord(ctx, updateRecordId, newData); err != nil {
		log.Fatalf("更新记录失败: %v", err)
	}
	fmt.Println("✓ 记录已更新 (RecordId=2)")
	
	// 验证更新
	updated, _ := recordStore.GetRecord(ctx, updateRecordId)
	fmt.Printf("  更新后的数据: %s\n", string(updated))
	
	// 12. 删除文档
	fmt.Println("\n=== 步骤 12: 删除文档 ===")
	deleteRecordId := storage.NewRecordIdFromLong(5)
	
	// 先从索引中删除
	deleteIdKey := []byte("5")
	idIndex.Remove(ctx, deleteIdKey, deleteRecordId)
	
	deleteNameKey := []byte("Eve")
	nameIndex.Remove(ctx, deleteNameKey, deleteRecordId)
	
	// 从 RecordStore 删除
	if err := recordStore.DeleteRecord(ctx, deleteRecordId); err != nil {
		log.Fatalf("删除记录失败: %v", err)
	}
	fmt.Println("✓ 记录已删除 (RecordId=5)")
	
	// 验证删除
	_, err = recordStore.GetRecord(ctx, deleteRecordId)
	if err != nil {
		fmt.Println("  验证: 记录确实已被删除")
	}
	
	// 13. 统计信息
	fmt.Println("\n=== 步骤 13: 统计信息 ===")
	stats := kvEngine.GetStats()
	fmt.Printf("✓ KV 引擎统计:\n")
	fmt.Printf("  - 运行状态: %v\n", stats["running"])
	fmt.Printf("  - RecordStore 数量: %v\n", stats["record_stores"])
	fmt.Printf("  - 索引数量: %v\n", stats["indexes"])
	fmt.Printf("  - 总记录数: %v\n", stats["total_records"])
	fmt.Printf("  - 总数据大小: %v 字节\n", stats["total_data_size"])
	fmt.Printf("  - 总索引条目: %v\n", stats["total_index_entries"])
	fmt.Printf("  - 活动会话数: %v\n", stats["sessions"])
	fmt.Printf("  - 创建的总会话数: %v\n", stats["total_sessions_created"])
	
	fmt.Println("\n=== RecordStore 统计 ===")
	fmt.Printf("  - 记录数: %d\n", recordStore.NumRecords())
	fmt.Printf("  - 数据大小: %d 字节\n", recordStore.DataSize())
	
	fmt.Println("\n=== 索引统计 ===")
	fmt.Printf("  - _id 索引条目数: %d\n", idIndex.NumEntries())
	fmt.Printf("  - name 索引条目数: %d\n", nameIndex.NumEntries())
	
	// 14. 事务回滚示例
	fmt.Println("\n=== 步骤 14: 事务回滚示例 ===")
	session2, _ := kvEngine.CreateSession(ctx)
	defer session2.End(ctx)
	
	session2.BeginTransaction(ctx)
	
	// 尝试插入一条记录
	tempRecordId := storage.NewRecordIdFromLong(999)
	tempData := []byte(`{"_id":999,"name":"Temp","age":99}`)
	recordStore.InsertRecord(ctx, tempRecordId, tempData)
	fmt.Println("✓ 插入临时记录 (RecordId=999)")
	
	// 回滚事务
	session2.RollbackTransaction(ctx)
	fmt.Println("✓ 事务已回滚")
	
	// 验证回滚（注意：简化实现中回滚可能不完全工作，需要完整的 Change 跟踪）
	fmt.Println("  注意: 简化实现中，已插入的数据不会自动回滚")
	fmt.Println("  完整实现需要在 RecordStore 中集成 RecoveryUnit 的 Change 机制")
	
	fmt.Println("\n=== 演示完成 ===")
	fmt.Println("✅ 所有存储引擎核心功能已演示")
}
