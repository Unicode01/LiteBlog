// Package utils provides utility functions for type conversion
package utils

import (
	"fmt"
	"strconv"
)

// GetBytesSafe safely converts any type to []byte
func GetBytesSafe(data any) []byte {
	switch v := data.(type) {
	case []byte: // []byte 和 []uint8 在 Go 中是相同类型
		return v
	case string:
		return []byte(v)
	default:
		return fmt.Appendf(nil, "%v", data)
	}
}

// GetStringSafe safely converts any type to string
func GetStringSafe(data any) string {
	switch v := data.(type) {
	case string:
		return v
	case []byte: // []byte 和 []uint8 在 Go 中是相同类型
		return string(v)
	default:
		return fmt.Sprintf("%v", data)
	}
}

// GetIntSafe safely converts any type to int
func GetIntSafe(data any) int {
	switch v := data.(type) {
	case int:
		return v
	case int8:
		return int(v)
	case int16:
		return int(v)
	case int32:
		return int(v)
	case int64:
		return int(v)
	case uint:
		return int(v)
	case uint8:
		return int(v)
	case uint16:
		return int(v)
	case uint32:
		return int(v)
	case uint64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	case []byte:
		// 处理 gRPC 插件返回的 []byte 类型数据
		if i, err := strconv.Atoi(string(v)); err == nil {
			return i
		}
		return 0
	case string:
		// 处理字符串类型的数字
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
		return 0
	default:
		return 0
	}
}

// GetInt64Safe safely converts any type to int64
func GetInt64Safe(data any) int64 {
	switch v := data.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case uint64:
		return int64(v)
	case float64:
		return int64(v)
	case []byte:
		// 处理 gRPC 插件返回的 []byte 类型数据
		if i, err := strconv.ParseInt(string(v), 10, 64); err == nil {
			return i
		}
		return 0
	case string:
		// 处理字符串类型的数字
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
		return 0
	default:
		return 0
	}
}
