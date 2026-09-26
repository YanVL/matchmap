package match

import (
	"context"
)

type Notifier interface {
	NotifyInviteCreated(ctx context.Context, userID string, senderID string, inviteID string) error
	NotifyMatchAccepted(ctx context.Context, userID, opponentID, matchID string) error
	NotifyMatchRejected(ctx context.Context, userID, inviteID string) error
	NotifyMatchFinished(ctx context.Context, userID, opponentID, matchID string) error
	NotifyMatchResult(ctx context.Context, recipientID, playerID, matchID, matchResult string) error
}
