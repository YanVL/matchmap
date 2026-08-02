package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"matchmap/internal/location"
)

type Handler struct {
	Hub *Hub
    LocationService *location.Service
}

// Handle chat message events
func (h *Handler) handleChatMessage(ctx context.Context, client *Client, event Event) {
	var chatMessage ChatMessage

	err := json.Unmarshal(event.Payload, &chatMessage)
	if err != nil {
		log.Printf("failed to unmarshal message: %v", err)
		return
	}

	msg := Message{
		User:    client.UserID,
		Content: chatMessage.Content,
		SentAt:  time.Now(),
	}

	h.Hub.Broadcast(ctx, msg)
}

// Handle location update events
func (h *Handler) handleLocationUpdate(ctx context.Context, client *Client, event Event) {
	var locationUpdate LocationUpdate

	err := json.Unmarshal(event.Payload, &locationUpdate)
	if err != nil {
		log.Printf("failed to unmarshal location update: %v", err)
		return
	}

	client.Location = Coordinates{
		Latitude:  locationUpdate.Latitude,
		Longitude: locationUpdate.Longitude,
	}

	err = h.LocationService.UpdateLocation(ctx, client.UserID, locationUpdate.Latitude, locationUpdate.Longitude)
	if err != nil {
		log.Printf("failed to update location in database: %v", err)
		return
	}

	log.Printf(
		"received location update: user_id=%s latitude=%f longitude=%f",
		client.UserID,
		locationUpdate.Latitude,
		locationUpdate.Longitude,
	)
}

// Dispatch events based on their type
func (h *Handler) dispatchEvent(ctx context.Context, client *Client, event Event) {
	switch event.Type {
	case "chat_message":
		h.handleChatMessage(ctx, client, event)
	case "location_update":
		h.handleLocationUpdate(ctx, client, event)
	default:
		log.Printf("unknown event type: %s", event.Type)
	}
}

// Start background tasks
func (h *Handler) startHeartbeat(ctx context.Context, cancel context.CancelFunc, client *Client) {
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				err := client.Conn.Ping(ctx)
				if err != nil {
					log.Printf("failed to send ping: %v", err)
					cancel()
					return
				}
			case <-ctx.Done():
				log.Printf(
					"heartbeat stopped: user_id=%s connection=%p",
					client.UserID,
					client.Conn,
				)
				return
			}
		}
	}()
}

// Process incoming events
func (h *Handler) readLoop(ctx context.Context, cancel context.CancelFunc, client *Client) {
	for {
		var event Event

		err := wsjson.Read(ctx, client.Conn, &event)
		if err != nil {
			log.Printf("failed to read event: %v", err)
			cancel()
			return
		}

		h.dispatchEvent(ctx, client, event)
	}
}

func (h *Handler) Connect(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Printf("failed to accept websocket connection: %v", err)
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer conn.CloseNow()
	defer cancel()

	client := &Client{
		UserID: r.URL.Query().Get("user"),
		Conn:   conn,
	}

	h.Hub.Register(client)
	defer h.Hub.Unregister(client)

	log.Printf(
		"client connected: user_id=%s active_clients=%d connection=%p",
		client.UserID,
		len(h.Hub.Clients),
		conn,
	)

	msgWelcome := Message{
		User:    "System",
		Content: "Welcome to the chat!",
		SentAt:  time.Now(),
	}

	err = wsjson.Write(ctx, client.Conn, msgWelcome)
	if err != nil {
		log.Printf("failed to send welcome message: %v", err)
	}

	h.startHeartbeat(ctx, cancel, client)

	h.readLoop(ctx, cancel, client)

}
