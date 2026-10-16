package websocket

import (
	"context"
	"matchmap/internal/location"
	"sync"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type Client struct {
	UserID          string
	Conn            *websocket.Conn
	Location        location.Coordinates
	LocationUpdated bool

	writeMu sync.Mutex
}

func (c *Client) WriteJSON(ctx context.Context, v any) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	return wsjson.Write(ctx, c.Conn, v)
}
