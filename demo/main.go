package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ansmatterer/socketioclient"
)

func main() {

	// 连接v2服务端
	clientV2, err := socketioclient.NewClient(socketioclient.Config{
		Version:        socketioclient.V2,
		Host:           "http://localhost:3000",
		Path:           "socket.io",
		Reconnect:      true,
		ReconnectDelay: 1 * time.Second,
	})
	if err != nil {
		fmt.Println("链接服务器错误:", err)
		return
	}
	clientV2.On("message", func(data []byte) {
		fmt.Printf("接收事件消息: %v\n", data)
	})

	// 发送消息
	clientV2.Emit("message", "Hello", func(response []byte) {
		fmt.Println("Server ACK:", response)
	})

	// 创建一个信号通道，用于接收操作系统发送的信号
	quit := make(chan os.Signal, 1)

	// 配置信号监听，关注中断信号和终止信号
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 阻塞等待信号，直到接收到指定的信号
	<-quit

	// 当接收到信号后，等待5秒后输出退出信息
	//time.Sleep(5 * time.Second)
	fmt.Println("退出程序！")

}
