package websocket

import (
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

	defer conn.CloseNow()

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

	err = wsjson.Write(r.Context(), client.Conn, msgWelcome)
	if err != nil {
		log.Printf("failed to send welcome message: %v", err)
	}

	for {
		var event Event

		err := wsjson.Read(r.Context(), conn, &event)
		if err != nil {
			log.Printf("failed to read event: %v", err)
			break
		}

		switch event.Type {
		case "chat_message":
			var chatMessage ChatMessage

			err := json.Unmarshal(event.Payload, &chatMessage)
			if err != nil {
				log.Printf("failed to unmarshal message: %v", err)
			}

			msg := Message{
				User:    client.UserID,
				Content: chatMessage.Content,
				SentAt:  time.Now(),
			}

			h.Hub.Broadcast(r.Context(), msg)
		default:
			log.Printf("unknown event type: %s", event.Type)
		}

	}
}
