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

type PrivateMessageNotification struct {
	ConversationID string `json:"conversation_id"`
	SenderID       string `json:"sender_id"`
	Content        string `json:"content"`
}
