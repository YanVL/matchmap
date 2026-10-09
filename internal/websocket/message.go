package websocket

import (
	"time"
)

type Message struct {
	User    string    `json:"user"`
	Content string    `json:"content"`
	SentAt  time.Time `json:"sent_at"`
}

type ChatMessage struct {
	Content string `json:"content"`
}
