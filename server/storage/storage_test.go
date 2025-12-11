package storage_test

import (
	"context"
	"testing"
	
	"github.com/zhukovaskychina/xmongodb/server/storage"
)

// TestKVEngine 测试 KV 引擎
func TestKVEngine(t *testing.T) {
	ctx := context.Background()
	
	// 创建 KV 引擎
	config := storage.KVEngineConfig{
		CacheSize:   1024 * 1024, // 1MB
		MaxSessions: 10,
	}
	engine := storage.NewKVEngine(config)
	
	// 启动引擎
	if err := engine.Start(ctx); err != nil {
		t.Fatalf("启动引擎失败: %v", err)
	}
	defer engine.Stop(ctx)
	
	t.Run("创建会话", func(t *testing.T) {
		session, err := engine.CreateSession(ctx)
		if err != nil {
			t.Fatalf("创建会话失败: %v", err)
		}
		
		if !session.IsActive() {
			t.Error("会话应该处于活动状态")
		}
		
		session.End(ctx)
	})
	
	t.Run("RecordStore操作", func(t *testing.T) {
		namespace := "test.collection"
		
		// 创建 RecordStore
		rs, err := engine.CreateRecordStore(namespace)
		if err != nil {
			t.Fatalf("创建 RecordStore 失败: %v", err)
		}
		
		// 插入记录
		recordId := storage.NewRecordIdFromLong(1)
		data := []byte(`{"name":"Alice","age":30}`)
		
		if err := rs.InsertRecord(ctx, recordId, data); err != nil {
			t.Fatalf("插入记录失败: %v", err)
		}
		
		// 读取记录
		retrieved, err := rs.GetRecord(ctx, recordId)
		if err != nil {
			t.Fatalf("读取记录失败: %v", err)
		}
		
		if string(retrieved) != string(data) {
			t.Errorf("记录数据不匹配: got %s, want %s", retrieved, data)
		}
		
		// 统计信息
		if rs.NumRecords() != 1 {
			t.Errorf("记录数不正确: got %d, want 1", rs.NumRecords())
		}
	})
	
	t.Run("索引操作", func(t *testing.T) {
		namespace := "test.indexed_collection"
		indexName := "name_idx"
		
		// 创建索引
		idx, err := engine.CreateSortedDataInterface(namespace, indexName, false)
		if err != nil {
			t.Fatalf("创建索引失败: %v", err)
		}
		
		// 插入索引条目
		key := []byte("Alice")
		recordId := storage.NewRecordIdFromLong(100)
		
		if err := idx.Insert(ctx, key, recordId); err != nil {
			t.Fatalf("插入索引条目失败: %v", err)
		}
		
		// 查找索引
		cursor, err := idx.Seek(ctx, key)
		if err != nil {
			t.Fatalf("查找索引失败: %v", err)
		}
		defer cursor.Close()
		
		if !cursor.Next() {
			t.Fatal("索引游标应该有数据")
		}
		
		foundRecordId := cursor.RecordId()
		if foundRecordId.Compare(recordId) != 0 {
			t.Errorf("RecordId 不匹配")
		}
	})
}

// TestRecoveryUnit 测试事务恢复单元
func TestRecoveryUnit(t *testing.T) {
	ctx := context.Background()
	
	ru := storage.NewRecoveryUnit()
	
	t.Run("事务提交", func(t *testing.T) {
		// 开始事务
		if err := ru.BeginTransaction(ctx); err != nil {
			t.Fatalf("开始事务失败: %v", err)
		}
		
		if !ru.IsActive() {
			t.Error("事务应该处于活动状态")
		}
		
		// 注册变更
		committed := false
		change := storage.NewSimpleChange(
			func() error {
				committed = true
				return nil
			},
			func() error {
				committed = false
				return nil
			},
		)
		
		if err := ru.RegisterChange(change); err != nil {
			t.Fatalf("注册变更失败: %v", err)
		}
		
		// 提交事务
		if err := ru.Commit(ctx); err != nil {
			t.Fatalf("提交事务失败: %v", err)
		}
		
		if !committed {
			t.Error("变更应该已提交")
		}
		
		if !ru.IsCommitted() {
			t.Error("事务应该已提交")
		}
	})
	
	t.Run("事务回滚", func(t *testing.T) {
		ru := storage.NewRecoveryUnit()
		
		// 开始事务
		if err := ru.BeginTransaction(ctx); err != nil {
			t.Fatalf("开始事务失败: %v", err)
		}
		
		// 注册变更
		rolledBack := false
		change := storage.NewSimpleChange(
			func() error { return nil },
			func() error {
				rolledBack = true
				return nil
			},
		)
		
		if err := ru.RegisterChange(change); err != nil {
			t.Fatalf("注册变更失败: %v", err)
		}
		
		// 回滚事务
		if err := ru.Rollback(ctx); err != nil {
			t.Fatalf("回滚事务失败: %v", err)
		}
		
		if !rolledBack {
			t.Error("变更应该已回滚")
		}
		
		if !ru.IsAborted() {
			t.Error("事务应该已中止")
		}
	})
}

// TestBTreeRecordStore 测试 B+Tree 记录存储
func TestBTreeRecordStore(t *testing.T) {
	ctx := context.Background()
	
	rs := storage.NewRecordStore("test.btree")
	
	t.Run("插入和读取", func(t *testing.T) {
		// 插入多条记录
		for i := int64(1); i <= 100; i++ {
			recordId := storage.NewRecordIdFromLong(i)
			data := []byte("test data " + string(rune('0'+i%10)))
			
			if err := rs.InsertRecord(ctx, recordId, data); err != nil {
				t.Fatalf("插入记录 %d 失败: %v", i, err)
			}
		}
		
		// 验证记录数
		if rs.NumRecords() != 100 {
			t.Errorf("记录数不正确: got %d, want 100", rs.NumRecords())
		}
		
		// 读取特定记录
		recordId := storage.NewRecordIdFromLong(50)
		data, err := rs.GetRecord(ctx, recordId)
		if err != nil {
			t.Fatalf("读取记录失败: %v", err)
		}
		
		if len(data) == 0 {
			t.Error("读取的数据不应为空")
		}
	})
	
	t.Run("扫描记录", func(t *testing.T) {
		cursor, err := rs.Scan(ctx, storage.NullRecordId())
		if err != nil {
			t.Fatalf("创建游标失败: %v", err)
		}
		defer cursor.Close()
		
		count := 0
		for cursor.Next() {
			count++
			recordId := cursor.RecordId()
			data := cursor.Data()
			
			if recordId.IsNull() {
				t.Error("RecordId 不应为空")
			}
			if len(data) == 0 {
				t.Error("数据不应为空")
			}
		}
		
		if count != 100 {
			t.Errorf("扫描记录数不正确: got %d, want 100", count)
		}
	})
	
	t.Run("更新和删除", func(t *testing.T) {
		recordId := storage.NewRecordIdFromLong(1)
		
		// 更新记录
		newData := []byte("updated data")
		if err := rs.UpdateRecord(ctx, recordId, newData); err != nil {
			t.Fatalf("更新记录失败: %v", err)
		}
		
		// 验证更新
		data, err := rs.GetRecord(ctx, recordId)
		if err != nil {
			t.Fatalf("读取记录失败: %v", err)
		}
		
		if string(data) != string(newData) {
			t.Errorf("数据不匹配: got %s, want %s", data, newData)
		}
		
		// 删除记录
		if err := rs.DeleteRecord(ctx, recordId); err != nil {
			t.Fatalf("删除记录失败: %v", err)
		}
		
		// 验证删除
		_, err = rs.GetRecord(ctx, recordId)
		if err == nil {
			t.Error("应该返回错误，因为记录已被删除")
		}
	})
}

// TestSortedDataInterface 测试索引接口
func TestSortedDataInterface(t *testing.T) {
	ctx := context.Background()
	
	t.Run("非唯一索引", func(t *testing.T) {
		idx := storage.NewSortedDataInterface("test_idx", false)
		
		// 插入相同键的多个条目
		key := []byte("duplicate_key")
		for i := int64(1); i <= 5; i++ {
			recordId := storage.NewRecordIdFromLong(i)
			if err := idx.Insert(ctx, key, recordId); err != nil {
				t.Fatalf("插入索引条目失败: %v", err)
			}
		}
		
		// 查找
		cursor, err := idx.Seek(ctx, key)
		if err != nil {
			t.Fatalf("查找失败: %v", err)
		}
		defer cursor.Close()
		
		count := 0
		for cursor.Next() {
			count++
		}
		
		if count != 5 {
			t.Errorf("找到的条目数不正确: got %d, want 5", count)
		}
	})
	
	t.Run("唯一索引", func(t *testing.T) {
		idx := storage.NewSortedDataInterface("unique_idx", true)
		
		key := []byte("unique_key")
		recordId1 := storage.NewRecordIdFromLong(1)
		recordId2 := storage.NewRecordIdFromLong(2)
		
		// 插入第一个条目
		if err := idx.Insert(ctx, key, recordId1); err != nil {
			t.Fatalf("插入第一个条目失败: %v", err)
		}
		
		// 尝试插入相同键（应该失败）
		err := idx.Insert(ctx, key, recordId2)
		if err == nil {
			t.Error("唯一索引应该拒绝重复键")
		}
	})
	
	t.Run("范围查询", func(t *testing.T) {
		idx := storage.NewSortedDataInterface("range_idx", false)
		
		// 插入有序数据
		for i := 0; i < 100; i++ {
			key := []byte{byte(i)}
			recordId := storage.NewRecordIdFromLong(int64(i))
			if err := idx.Insert(ctx, key, recordId); err != nil {
				t.Fatalf("插入失败: %v", err)
			}
		}
		
		// 范围查询 [10, 20)
		startKey := []byte{10}
		endKey := []byte{20}
		
		cursor, err := idx.SeekRange(ctx, startKey, endKey)
		if err != nil {
			t.Fatalf("范围查询失败: %v", err)
		}
		defer cursor.Close()
		
		count := 0
		for cursor.Next() {
			count++
		}
		
		if count != 10 {
			t.Errorf("范围查询结果数不正确: got %d, want 10", count)
		}
	})
}

// BenchmarkRecordStoreInsert 基准测试：插入记录
func BenchmarkRecordStoreInsert(b *testing.B) {
	ctx := context.Background()
	rs := storage.NewRecordStore("bench.collection")
	data := []byte("benchmark data for testing insert performance")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recordId := storage.NewRecordIdFromLong(int64(i))
		if err := rs.InsertRecord(ctx, recordId, data); err != nil {
			b.Fatalf("插入失败: %v", err)
		}
	}
}

// BenchmarkRecordStoreGet 基准测试：读取记录
func BenchmarkRecordStoreGet(b *testing.B) {
	ctx := context.Background()
	rs := storage.NewRecordStore("bench.collection")
	data := []byte("benchmark data")
	
	// 预先插入数据
	for i := 0; i < 10000; i++ {
		recordId := storage.NewRecordIdFromLong(int64(i))
		rs.InsertRecord(ctx, recordId, data)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recordId := storage.NewRecordIdFromLong(int64(i % 10000))
		if _, err := rs.GetRecord(ctx, recordId); err != nil {
			b.Fatalf("读取失败: %v", err)
		}
	}
}

// BenchmarkIndexInsert 基准测试：插入索引
func BenchmarkIndexInsert(b *testing.B) {
	ctx := context.Background()
	idx := storage.NewSortedDataInterface("bench_idx", false)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := []byte{byte(i % 256), byte(i / 256)}
		recordId := storage.NewRecordIdFromLong(int64(i))
		if err := idx.Insert(ctx, key, recordId); err != nil {
			b.Fatalf("插入失败: %v", err)
		}
	}
}
