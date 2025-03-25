package socketioclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn            *websocket.Conn //websocket连接
	config          Config          //配置
	sid             string
	pingInterval    time.Duration
	pingTimeout     time.Duration
	eventHandlers   map[string]func([]byte)
	ackCallbacks    map[int]func([]byte)
	writeChan       chan []byte
	closeChan       chan struct{}
	connChan        chan []byte //接收链接信息的通道
	heartbeatChan   chan string //心跳管理
	heartbeatCtx    context.Context
	heartbeatCancel context.CancelFunc
	reconnect       *reconnectManager
	mutex           sync.RWMutex
}

func (c *Client) resetTimeout() {
	if c.heartbeatCancel != nil {
		c.heartbeatCancel() // 先取消旧的 context
	}
	c.heartbeatCtx, c.heartbeatCancel = context.WithTimeout(context.Background(), c.pingTimeout)
	fmt.Printf("%d秒后超时\n", c.pingTimeout/time.Second)
}

func NewClient(cfg Config) (*Client, error) {
	if cfg.Host == "" {
		return nil, errors.New("host cannot be empty")
	}

	client := &Client{
		config:        cfg,
		eventHandlers: make(map[string]func([]byte)),
		ackCallbacks:  make(map[int]func([]byte)),
		writeChan:     make(chan []byte, 128),
		connChan:      make(chan []byte, 128),
		reconnect:     newReconnectManager(cfg),
	}

	if err := client.connect(); err != nil {
		return nil, err
	}
	fmt.Printf("链接服务器成功%v\n", client)

	return client, nil
}

func (c *Client) connect() error {
	// WebSocket连接
	wsURL := c.handshakeURL() //fmt.Sprintf("%s://%s/%s/?EIO=%s&transport=websocket",scheme, u.Host, c.config.Path, eioVersions[c.config.Version])

	// WebSocket连接
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrWebsocketFailed, err)
	}
	//log.Println("WebSocket连接成功")
	//log.Printf("c.closeChan:%v", c.closeChan)
	c.conn = conn
	c.closeChan = make(chan struct{})
	c.heartbeatChan = make(chan string)
	go c.readPump()
	go c.writePump()

	fmt.Println("等待连接响应")
	received := <-c.connChan
	fmt.Printf("等待连接响应:%v\n", string(received))
	c.OpenInit(received)
	//监测心跳
	go c.heartbeat()
	//c.emitEvent("connect", nil)
	return nil
}

// 心跳管理
func (c *Client) heartbeat() {
	ticker := time.NewTicker(c.pingInterval)
	defer ticker.Stop()
	c.resetTimeout()
	for {
		select {
		case <-ticker.C:
			log.Println("ping：", _enginePingPacket)
			c.write([]byte(_enginePingPacket)) // ping
		case <-c.closeChan:
			log.Println("关闭心跳")
			return
		case <-c.heartbeatCtx.Done():
			//心跳监测超时
			log.Println("心跳超时")
			//关闭连接信息准备重连
			c.Close(false)
			//重连
			c.reconnect.trigger(c)
			return
		case <-c.heartbeatChan:
			//回复了重新计时
			c.resetTimeout()

		}
	}
}

// 写消息
func (c *Client) write(message []byte) error {
	select {
	case c.writeChan <- message:
		return nil
	case <-time.After(5 * time.Second):
		return ErrWriteTimeout
	}
}

// 核心写循环
func (c *Client) writePump() {
	for {
		select {
		case msg := <-c.writeChan:
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Printf("Write error: %v", err)
			}
		case <-c.closeChan:
			log.Println("关闭写循环")
			return
		}
	}
}

// 链接响应,发起心跳
func (c *Client) OpenInit(resp []byte) {
	//{"sid":"a721642c-b2c8-431f-81b6-e264e01c9167","upgrades":["websocket"],"pingInterval":25000,"pingTimeout":600000}
	var hs handshakeResponse
	reader := bytes.NewReader(resp)
	if err := json.NewDecoder(reader).Decode(&hs); err != nil {
		log.Printf("Decode error: %v", err)
	}
	c.sid = hs.SID
	c.config.Timeout = time.Duration(hs.PingTimeout) * time.Millisecond
	c.pingInterval = time.Duration(hs.PingInterval) * time.Millisecond
	c.pingTimeout = time.Duration(hs.PingTimeout) * time.Millisecond
}

// 核心读循环
func (c *Client) readPump() {
	//defer c.Close()
	for {
		_, message, err := c.conn.ReadMessage()
		fmt.Printf("Received message: %s\n", string(message))
		if err != nil {
			/*log.Printf("Read error: %v", err)
			fmt.Printf("链接断开%v\n", websocket.IsUnexpectedCloseError(err))
			if websocket.IsUnexpectedCloseError(err) && c.config.Reconnect {
				c.reconnect.trigger(c)
			}*/

			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				log.Println("Server closed the connection normally.")
				//c.reconnect.trigger(c)
			} else if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				//c.reconnect.trigger(c)
				log.Println("Server closed unexpectedly:", err)
			} else {
				log.Println("Read error:", err)
			}
			//关闭连接
			c.Close(false)
			//重连
			c.reconnect.trigger(c)
			log.Println("重连成功...")
			return
		}

		c.handleMessage(message)
	}
}

// 事件处理
func (c *Client) On(event string, handler func([]byte)) {
	c.eventHandlers[event] = handler
}

// 写入事件
func (c *Client) Emit(event string, data interface{}, ack func([]byte)) error {
	packet := []interface{}{event, data}
	packetBytes, _ := json.Marshal(packet)

	var msg []byte
	var err error
	if ack != nil {
		id := c.reconnect.getAckID()
		//msg = fmt.Sprintf("%d%s", id, packetBytes)
		msg = append([]byte(fmt.Sprintf("%d", id)), packetBytes...)
		c.ackCallbacks[id] = ack
	}
	// 对消息进行编码，以便通过底层的传输协议发送。
	msg, err = encodeSocketPacket(_socketEventPacket, msg)
	if err != nil {
		return err
	}
	// 对消息进行编码，以便通过底层的传输协议发送。
	msg, err = encodeEnginePacket(_engineMessagePacket, msg)
	if err != nil {
		return err
	}

	c.write(msg)
	return nil
}

func (c *Client) Close(sendClose ...bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	//关闭写循环
	log.Println("关闭资源")
	close(c.closeChan)
	c.conn.Close()
	//传了false才不执行关闭事件
	if len(sendClose) == 0 || sendClose[0] {
		c.emitEvent(_socketDisconnectPacket, _engineClosePacket)
	} else {
		log.Println("不发送关闭事件")
	}

}

// 发送特点协议事件
func (c *Client) emitEvent(socket string, engine string) {
	msg := fmt.Sprintf("%s%s", engine, socket)
	c.write([]byte(msg))
}
