package chat

import (
	"context"
)

type Notifier interface {
	NotifyMessageReceived(ctx context.Context, recipientID, conversationID, senderID, message string) error
}
