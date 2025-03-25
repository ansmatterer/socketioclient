package socketioclient

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
)

// 处理消息
func (c *Client) handleMessage(message []byte) {
	var once sync.Once // 确保只发送一次
	packetType, payload, err := decodeEnginePacket(message)
	log.Printf("handleMessage: %s", packetType)
	if err != nil {
		log.Printf("Decode error: %v", err)
		return
	}
	switch packetType {
	case _engineMessagePacket: // 消息 4
		if len(payload) == 0 {
			log.Printf("empty message")
			return
		}
		c.socketMessage(payload)
	case _engineClosePacket: // 关闭 1
		c.Close()
		return
	case _enginePongPacket: // 心跳响应 3
		c.heartbeatChan <- packetType
		return
	case _enginePingPacket: // 心跳 2
		//响应心跳
		c.write([]byte(_enginePongPacket))
	case _engineOpenPacket: //链接响应 0
		once.Do(func() {
			fmt.Printf("[协程] 发送链接响应消息到主程序\n")
			c.connChan <- payload
		})
		//c.OpenInit(payload)
	}

}

// 处理socket.io层消息
func (c *Client) socketMessage(message []byte) {
	packetType, payload, err := decodeSocketPacket(message)
	log.Printf("socketMessage: %s", packetType)
	if err != nil {
		log.Printf("Decode error: %v", err)
		return
	}
	switch packetType {
	case _socketEventPacket: //事件 2
		c.eventHandlersMessage(payload)
		return
	case _socketAckPacket: // 事件回调响应 3
		//处理事件消息
		c.handlEventMessage(payload)
	case _socketErrorPacket: // 错误 4
	}
}

// 根据消息获取事件ID和消息内容
func getEventMessage(message []byte) (int, []byte) {
	var packetType []byte
	var key int = 0
	//获取message字符串前面的数字前缀
	for i, v := range message {
		//找到第一个非数字字符
		if v < '0' || v > '9' {
			key = i
			break
		}
		packetType = append(packetType, v)

	}
	message = message[key:]
	//将packetType转为整数
	eventID, err := strconv.Atoi(string(packetType))
	if err != nil {
		return 0, message
	}
	return eventID, message
}

// 处理事件消息
func (c *Client) handlEventMessage(message []byte) {
	eventID, message := getEventMessage(message)
	if eventID == 0 {
		//处理公共事件消息
		c.eventHandlersMessage(message)
	} else {
		//处理回调事件响应消息
		c.callbakfun(eventID, message)
	}
}

// 执行回调函数
func (c *Client) callbakfun(eventID int, message []byte) {
	if callback, exists := c.ackCallbacks[eventID]; exists {
		callback(message)
		delete(c.ackCallbacks, eventID)
	}
}
func (c *Client) eventHandlersMessage(message []byte) {
	//将消息转为json串
	event, msg := GetEvent(message)
	if event != "" {
		if callback, exists := c.eventHandlers[event]; exists {
			callback(msg)
			delete(c.eventHandlers, event)
		}
	}
}

func GetEvent(data []byte) (string, []byte) {
	var arr []interface{}
	if err := json.Unmarshal(data, &arr); err != nil {
		return "", data
	}
	//arr[1]转换成[]byte
	data, _ = json.Marshal(arr[1])
	if len(arr) >= 2 {
		if event, ok := arr[0].(string); ok {
			return event, data
		}
	}
	return "", data
}
