package websocket

import (
	"context"
	"log"

	"github.com/coder/websocket/wsjson"
)

type Hub struct {
    Clients map[string]*Client
}

func NewHub() *Hub {
    return &Hub{
        Clients: make(map[string]*Client),
    }
}

func (h *Hub) Register(client *Client) {
    h.Clients[client.UserID] = client
}

func (h *Hub) Unregister(client *Client) {
    delete(h.Clients, client.UserID)
}

func (h *Hub) Broadcast(ctx context.Context, message Message) {
    for _, client := range h.Clients {
        
        log.Printf("sending message to client %s: %+v", client.UserID, message)
        
        err := wsjson.Write(ctx, client.Conn, message)
        if err != nil {
            log.Printf("failed to send message to client %s: %v", client.UserID, err)
        }
    }
}