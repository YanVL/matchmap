package websocket

import (
	"github.com/coder/websocket"
)

type Client struct {
	UserID string
	Conn   *websocket.Conn
}
