package chat

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	Repository *Repository
}

var ErrUserNotInConversation = errors.New("user is not part of the conversation")
var ErrNoActiveMatch = errors.New("users do not have an active match or neither are friends")
var ErrEmptyMessage = errors.New("message content is empty")
var ErrMessageTooLong = errors.New("message content is too long")
var ErrUserIDRequired = errors.New("user ID is required")
var ErrSameUser = errors.New("users cannot be the same")
var ErrMessageIDsRequired = errors.New("conversation_id and sender_id are required")
var ErrInvalidUUID = errors.New("invalid UUID format")

func (s *Service) GetOrCreateConversation(ctx context.Context, user1ID, user2ID string) (string, error) {

	if user1ID == "" || user2ID == "" {
		return "", ErrUserIDRequired
	}

	if user1ID == user2ID {
		return "", ErrSameUser
	}

	if err := validateUUIDs(user1ID, user2ID); err != nil {
		return "", err
	}

	hasActiveMatch, err := s.Repository.HasActiveMatch(ctx, user1ID, user2ID)

	if err != nil {
		return "", err
	}

	if !hasActiveMatch {
		return "", ErrNoActiveMatch
	}

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

func (s *Service) CreateMessage(ctx context.Context, conversationID, senderID, content string) (string, error) {

	if conversationID == "" || senderID == "" {
		return "", ErrMessageIDsRequired
	}

	if strings.TrimSpace(content) == "" {
		return "", ErrEmptyMessage
	}

	if err := validateUUIDs(conversationID, senderID); err != nil {
		return "", err
	}

	if len([]rune(content)) > 1000 {
		return "", ErrMessageTooLong
	}

	user1ID, user2ID, err := s.Repository.GetConversationByID(ctx, conversationID)

	if err != nil {
		return "", err
	}

	if senderID != user1ID && senderID != user2ID {
		return "", ErrUserNotInConversation
	}

	hasActiveMatch, err := s.Repository.HasActiveMatch(ctx, user1ID, user2ID)

	if err != nil {
		return "", err
	}

	if !hasActiveMatch {
		return "", ErrNoActiveMatch
	}

	return s.Repository.CreateMessage(ctx, conversationID, senderID, content)
}

func validateUUIDs(ids ...string) error {
	for _, id := range ids {
		if _, err := uuid.Parse(id); err != nil {
			return ErrInvalidUUID
		}
	}

	return nil
}
