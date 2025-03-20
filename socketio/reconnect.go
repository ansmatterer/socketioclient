package socketio

import (
	"log"
	"sync"
	"time"
)

type reconnectManager struct {
	cfg        Config
	attempts   int
	ackCounter int
	mutex      sync.Mutex
}

func newReconnectManager(cfg Config) *reconnectManager {
	return &reconnectManager{
		cfg: cfg,
	}
}

// trigger 尝试重新连接客户端。
// 该函数首先检查当前的重连尝试次数是否超过了配置的最大尝试次数，如果没有，则计算下一次重连的延迟时间。
// 在延迟时间结束后，它将尝试重新连接客户端。如果重连失败，它将增加重连尝试次数并再次尝试重连。
// 如果重连成功，它将重置重连尝试次数。
// 参数:
//
//	c (*Client): 需要重新连接的客户端实例。
func (r *reconnectManager) trigger(c *Client) {
	// 加锁以确保并发安全。
	r.mutex.Lock()

	// 检查是否达到了最大重连尝试次数。
	if r.cfg.ReconnectAttempts > 0 && r.attempts >= r.cfg.ReconnectAttempts {
		log.Printf("Max reconnect attempts reached: %d", r.attempts)
		return
	}

	// 计算下一次重连的延迟时间。
	delay := r.calculateDelay()
	log.Printf("Reconnecting in %v (attempt %d)", delay, r.attempts+1)

	// 等待延迟时间。
	time.Sleep(delay)
	// 尝试重新连接客户端。
	if err := c.connect(); err != nil {
		log.Printf("Reconnect failed: %v", err)
		r.attempts++
		r.mutex.Unlock()
		r.trigger(c)
	} else {
		// 如果重连成功，重置重连尝试次数。
		r.attempts = 0
		r.mutex.Unlock()
		return
	}

}

func (r *reconnectManager) calculateDelay() time.Duration {
	base := float64(r.cfg.ReconnectDelay)
	maxDelay := 30 * time.Second
	delay := base * float64(r.attempts+1)
	if delay > float64(maxDelay) {
		return maxDelay
	}
	return time.Duration(delay)
}

func (r *reconnectManager) getAckID() int {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.ackCounter++
	return r.ackCounter
}
