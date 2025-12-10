// Package plugins provides configuration for plugin system
package plugins

import "time"

// PluginConfig 插件系统配置
type PluginConfig struct {
	// gRPC 相关配置
	GRPCListenerAddress string        // gRPC 监听地址，例如 "127.0.0.1:9090"
	CommandTimeout      time.Duration // 命令超时时间
	HeartbeatInterval   time.Duration // 心跳间隔

	// JavaScript 相关配置
	PluginDir     string        // 插件目录路径
	JSInitDelay   time.Duration // JS 插件初始化延迟
	EnableJSWatch bool          // 是否启用 JS 文件监视（热重载）

	// 通用配置
	MaxPlugins     int  // 最大插件数量，0 表示无限制
	EnableDebugLog bool // 是否启用调试日志
}

// DefaultConfig 返回默认配置
func DefaultConfig() *PluginConfig {
	return &PluginConfig{
		GRPCListenerAddress: "127.0.0.1:9090",
		CommandTimeout:      30 * time.Second,
		HeartbeatInterval:   30 * time.Second,
		PluginDir:           "plugins",
		JSInitDelay:         2 * time.Second,
		EnableJSWatch:       false,
		MaxPlugins:          0,
		EnableDebugLog:      false,
	}
}

// PluginInfo 插件元数据信息
type PluginInfo struct {
	ID          string   // 插件唯一标识
	Name        string   // 插件名称
	Version     string   // 插件版本
	Author      string   // 插件作者
	Description string   // 插件描述
	Priority    int      // 执行优先级（数字越小优先级越高）
	Methods     []string // 注册的方法列表
	LoadedAt    time.Time // 加载时间
}

// PluginStatus 插件状态
type PluginStatus int

const (
	PluginStatusUnknown   PluginStatus = iota // 未知状态
	PluginStatusLoading                       // 加载中
	PluginStatusRunning                       // 运行中
	PluginStatusStopped                       // 已停止
	PluginStatusError                         // 错误状态
)

func (s PluginStatus) String() string {
	switch s {
	case PluginStatusLoading:
		return "loading"
	case PluginStatusRunning:
		return "running"
	case PluginStatusStopped:
		return "stopped"
	case PluginStatusError:
		return "error"
	default:
		return "unknown"
	}
}

