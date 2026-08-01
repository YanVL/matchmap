package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type Handler struct {
	Hub *Hub
}

func (h Handler) Connect(w http.ResponseWriter, r *http.Request) {
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

	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				err := conn.Ping(ctx)
				if err != nil {
					log.Printf("failed to send ping: %v", err)
					cancel()
					return
				}
			case <-ctx.Done():
				log.Printf(
					"heartbeat stopped: user_id=%s connection=%p",
					client.UserID,
					conn,
				)
				return
			}
		}
	}()

	for {
		var event Event

		err := wsjson.Read(ctx, conn, &event)
		if err != nil {
			log.Printf("failed to read event: %v", err)
			cancel()
			break
		}

		switch event.Type {
		case "chat_message":
			var chatMessage ChatMessage

			err := json.Unmarshal(event.Payload, &chatMessage)
			if err != nil {
				log.Printf("failed to unmarshal message: %v", err)
				continue
			}

			msg := Message{
				User:    client.UserID,
				Content: chatMessage.Content,
				SentAt:  time.Now(),
			}

			h.Hub.Broadcast(ctx, msg)
		case "location_update":

			var locationUpdate LocationUpdate

			err := json.Unmarshal(event.Payload, &locationUpdate)
			if err != nil {
				log.Printf("failed to unmarshal location update: %v", err)
				continue
			}

			client.Location = Coordinates{
				Latitude:  locationUpdate.Latitude,
				Longitude: locationUpdate.Longitude,
			}

			log.Printf(
				"received location update: user_id=%s latitude=%f longitude=%f",
				client.UserID,
				locationUpdate.Latitude,
				locationUpdate.Longitude,
			)

			// h.Hub.UpdateClientLocation(client)
		default:
			log.Printf("unknown event type: %s", event.Type)
		}
	}

}
