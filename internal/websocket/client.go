package websocket

import (
	"github.com/coder/websocket"
)

type Client struct {
	UserID string
	UserName string
	Conn   *websocket.Conn
}
