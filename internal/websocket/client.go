package websocket

import (
	"github.com/coder/websocket"
	"matchmap/internal/location"
)

type Client struct {
	UserID   string
	Conn     *websocket.Conn
	Location location.Coordinates
}