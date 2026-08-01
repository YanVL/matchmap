package websocket

import (
	"github.com/coder/websocket"
)

type Client struct {
	UserID string
	Conn   *websocket.Conn
	Location Coordinates
}

type Coordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}