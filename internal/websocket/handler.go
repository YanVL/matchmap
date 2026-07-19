package websocket

import (
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
		UserID: "b34ac619-afe7-4321-a672-0366d9cec18b",
		Conn:   conn,
	}

	h.Hub.Register(client)

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
		_, _, err := conn.Read(r.Context())
		if err != nil {
			log.Printf("client connection ended: user_id=%s error=%v", client.UserID, err)
			h.Hub.Unregister(client)
			break
		}
	}
}
