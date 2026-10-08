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
