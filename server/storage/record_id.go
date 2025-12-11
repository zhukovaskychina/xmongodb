package storage

import (
	"encoding/binary"
	"fmt"
)

// RecordId 记录标识符
// 用于唯一标识一条记录，支持 int64 和 byte[] 两种形式
type RecordId struct {
	// repr 表示类型: 0=null, 1=int64, 2=bytes
	repr int8
	// long 存储 int64 类型的 RecordId
	long int64
	// data 存储 byte[] 类型的 RecordId
	data []byte
}

// NewRecordIdFromLong 从 int64 创建 RecordId
func NewRecordIdFromLong(id int64) RecordId {
	return RecordId{
		repr: 1,
		long: id,
	}
}

// NewRecordIdFromBytes 从 byte[] 创建 RecordId
func NewRecordIdFromBytes(data []byte) RecordId {
	copied := make([]byte, len(data))
	copy(copied, data)
	return RecordId{
		repr: 2,
		data: copied,
	}
}

// NullRecordId 返回空的 RecordId
func NullRecordId() RecordId {
	return RecordId{repr: 0}
}

// IsNull 检查是否为空
func (r RecordId) IsNull() bool {
	return r.repr == 0
}

// IsLong 检查是否为 int64 类型
func (r RecordId) IsLong() bool {
	return r.repr == 1
}

// AsLong 获取 int64 值
func (r RecordId) AsLong() (int64, bool) {
	if r.repr != 1 {
		return 0, false
	}
	return r.long, true
}

// AsBytes 获取 byte[] 值
func (r RecordId) AsBytes() ([]byte, bool) {
	if r.repr == 2 {
		return r.data, true
	}
	if r.repr == 1 {
		// 将 int64 转换为字节
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, uint64(r.long))
		return buf, true
	}
	return nil, false
}

// Compare 比较两个 RecordId
// 返回: -1 (小于), 0 (等于), 1 (大于)
func (r RecordId) Compare(other RecordId) int {
	if r.repr != other.repr {
		return int(r.repr - other.repr)
	}
	
	switch r.repr {
	case 0:
		return 0
	case 1:
		if r.long < other.long {
			return -1
		} else if r.long > other.long {
			return 1
		}
		return 0
	case 2:
		return compareBytes(r.data, other.data)
	}
	
	return 0
}

// String 字符串表示
func (r RecordId) String() string {
	switch r.repr {
	case 0:
		return "RecordId(null)"
	case 1:
		return fmt.Sprintf("RecordId(%d)", r.long)
	case 2:
		return fmt.Sprintf("RecordId(0x%x)", r.data)
	}
	return "RecordId(unknown)"
}

// compareBytes 比较两个字节数组
func compareBytes(a, b []byte) int {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}
	
	for i := 0; i < minLen; i++ {
		if a[i] < b[i] {
			return -1
		} else if a[i] > b[i] {
			return 1
		}
	}
	
	if len(a) < len(b) {
		return -1
	} else if len(a) > len(b) {
		return 1
	}
	
	return 0
}
