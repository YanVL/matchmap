package websocket

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