package socketio

import "errors"

var (
	ErrConnectionClosed = errors.New("connection closed")
	ErrHandshakeFailed  = errors.New("handshake failed")
	ErrInvalidPacket    = errors.New("invalid packet format")
	ErrWriteTimeout     = errors.New("write operation timed out")
	ErrWebsocketFailed  = errors.New("websocket failed")
)
