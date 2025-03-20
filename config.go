package socketioclient

import "time"

type Version string

const (
	V2 Version = "v2"
	V3 Version = "v3"
)

type Config struct {
	Version           Version       // 协议版本 v2/v3
	Host              string        // 服务器地址
	Path              string        // Socket.IO路径
	Reconnect         bool          // 启用自动重连
	ReconnectAttempts int           // 最大重连次数 (0=无限)
	ReconnectDelay    time.Duration // 基础重连间隔
	Timeout           time.Duration // 连接超时
}
