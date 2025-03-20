package socketio

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// Engine.IO 版本映射
var eioVersions = map[Version]string{
	V2: "3",
	V3: "4",
}

// 握手响应结构
type handshakeResponse struct {
	SID          string   `json:"sid"`
	Upgrades     []string `json:"upgrades"`
	PingTimeout  int      `json:"pingTimeout"`
	PingInterval int      `json:"pingInterval"`
}

// Engine.IO 协议包类型
const (
	_engineOpenPacket    = "0" // 连接
	_engineClosePacket   = "1" // 断开连接
	_enginePingPacket    = "2" // 心跳
	_enginePongPacket    = "3" // 心跳响应
	_engineMessagePacket = "4" // 消息
)

// Socket.IO 协议包类型
const (
	_socketConnectPacket    = "0"  //连接
	_socketDisconnectPacket = "1"  //断开连接
	_socketEventPacket      = "2"  // 事件
	_socketAckPacket        = "3"  // 事件响应
	_socketErrorPacket      = "4 " // 错误
)

// 生成握手URL
func (c *Client) handshakeURL() string {
	u, _ := url.Parse(c.config.Host)
	scheme := "ws"
	if u.Scheme == "https" {
		scheme = "wss"
	}
	query := fmt.Sprintf("EIO=%s&transport=websocket",
		eioVersions[c.config.Version])

	if c.config.Path != "" {
		return fmt.Sprintf("%s://%s/%s/?%s", scheme, u.Host, c.config.Path, query)
	}
	return fmt.Sprintf("%s://%s/?%s", scheme, u.Host, query)
}

// 解析Socket.IO消息包
func _encodePacket(packetType string, data interface{}) ([]byte, error) {
	var sb strings.Builder
	sb.WriteString(packetType)

	if data != nil {
		switch v := data.(type) {
		case string:
			sb.WriteString(v)
		case []byte:
			sb.Write(v)
		default:
			jsonData, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			sb.Write(jsonData)
		}
	}
	return []byte(sb.String()), nil
}

func _decodePacket(data []byte) (string, []byte, error) {

	if len(data) < 1 {
		return "0", nil, ErrInvalidPacket
	}

	packetType := string(data[0])

	var payload []byte
	if len(data) > 1 {

		payload = data[1:]
	}

	return packetType, payload, nil
}

func encodeSocketPacket(packetType string, data interface{}) ([]byte, error) {
	/*判断是不是以下类型*const (
		_socketConnectPacket    = "0"  //连接
		_socketDisconnectPacket = "1"  //断开连接
		_socketEventPacket      = "2"  // 事件
		_socketAckPacket        = "3"  // 事件响应
		_socketErrorPacket      = "4 " // 错误
	)*/

	return _encodePacket(packetType, data)
}

func decodeSocketPacket(data []byte) (string, []byte, error) {
	return _decodePacket(data)
}

func encodeEnginePacket(packetType string, data interface{}) ([]byte, error) {
	return _encodePacket(packetType, data)
}

func decodeEnginePacket(data []byte) (string, []byte, error) {
	return _decodePacket(data)
}
