package websocket

import (
	// "context"
	"net/http"
	// "time"
	"log"

	"github.com/coder/websocket"
)

type Handler struct{}

func (h Handler) Connect(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("New connection: %p", conn)

	defer conn.CloseNow()



}
