package match

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB *pgxpool.Pool
}

func (r *Repository) CreateInvite(ctx context.Context, sender, receiver string) error {
	_, err := r.DB.Exec(
		ctx,
		`
		INSERT INTO match_invites (sender_id, receiver_id, status, created_at)
		VALUES ($1, $2, 'pending', NOW())
		`,
		sender,
		receiver,
	)

	return err
}

func (r *Repository) IsInvited(ctx context.Context, sender, receiver string) (bool, error) {
	var exists bool
	err := r.DB.QueryRow(
		ctx,
		`
		SELECT EXISTS (
			SELECT 1 FROM match_invites
			WHERE status = 'pending'
				AND LEAST(sender_id, receiver_id) = LEAST($1::uuid, $2::uuid)
				AND GREATEST(sender_id, receiver_id) = GREATEST($1::uuid, $2::uuid)
			)
		`,
		sender,
		receiver,
	).Scan(&exists)

	return exists, err
}
