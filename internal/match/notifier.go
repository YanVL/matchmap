package match

import (
	"context"
)

type Notifier interface {
    NotifyInviteCreated(ctx context.Context, userID string, senderID string) error
}