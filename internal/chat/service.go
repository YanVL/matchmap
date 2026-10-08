package chat

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type Service struct {
	Repository *Repository
}

func (s *Service) GetOrCreateConversation(ctx context.Context, user1ID, user2ID string) (string, error) {
	conversationID, err := s.Repository.GetConversationByUsers(ctx, user1ID, user2ID)

	if err == nil {
		return conversationID, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	newConversationID, err := s.Repository.CreateConversation(ctx, user1ID, user2ID)
	if err == nil {
		return newConversationID, nil
	}

	if !errors.Is(err, ErrConversationConflict) {
		return "", err
	}

	return s.Repository.GetConversationByUsers(ctx, user1ID, user2ID)
}

