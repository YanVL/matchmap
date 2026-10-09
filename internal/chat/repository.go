package chat

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB *pgxpool.Pool
}

var ErrConversationConflict = errors.New("conversation already exists")

func (r *Repository) CreateConversation(ctx context.Context, user1ID, user2ID string) (string, error) {
	query := `
		INSERT INTO conversations (user_1_id, user_2_id)
		VALUES ($1, $2)
		RETURNING id
	`

	var conversationID string

	err := r.DB.QueryRow(ctx, query, user1ID, user2ID).Scan(&conversationID)
	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "conversations_unique_pair_idx" {
			return "", ErrConversationConflict
		}

		return "", err
	}
	return conversationID, nil
}

func (r *Repository) GetConversationByUsers(ctx context.Context, user1ID, user2ID string) (string, error) {
	query := `
		SELECT id FROM conversations
		WHERE LEAST(user_1_id, user_2_id) = LEAST($1::uuid, $2::uuid)
			AND GREATEST(user_1_id, user_2_id) = GREATEST($1::uuid, $2::uuid)
		`

	var conversationID string
	err := r.DB.QueryRow(ctx, query, user1ID, user2ID).Scan(&conversationID)
	if err != nil {
		return "", err
	}
	return conversationID, nil
}

func (r *Repository) GetConversationByID(ctx context.Context, conversationID string) (string, string, error) {
	query := `
		SELECT user_1_id, user_2_id FROM conversations
		WHERE id = $1
	`

	var user1ID, user2ID string
	err := r.DB.QueryRow(ctx, query, conversationID).Scan(&user1ID, &user2ID)
	if err != nil {
		return "", "", err
	}

	return user1ID, user2ID, nil
}

func (r *Repository) CreateMessage(ctx context.Context, conversationID, senderID, content string) (string, error) {
	query := `
		INSERT INTO messages (conversation_id, sender_id, content, created_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id
	`

	var messageID string
	err := r.DB.QueryRow(ctx, query, conversationID, senderID, content).Scan(&messageID)
	if err != nil {
		return "", err
	}
	return messageID, nil
}

func (r *Repository) HasActiveMatch(ctx context.Context, user1ID, user2ID string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM matches
			WHERE status = 'active'
			AND (
				(player_1_id = $1 AND player_2_id = $2)
				OR
				(player_1_id = $2 AND player_2_id = $1)
			)
		)
	`

	var exists bool
	err := r.DB.QueryRow(ctx, query, user1ID, user2ID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}
