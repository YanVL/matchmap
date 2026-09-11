package match

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB *pgxpool.Pool
}

var ErrInviteConflict = errors.New("invite already exists")
var ErrInviteNotFound = errors.New("invite not found")

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

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) &&
			pgErr.Code == "23505" &&
			pgErr.ConstraintName == "match_invites_pending_pair_idx" {
			return ErrInviteConflict
		}

		return err
	}

	return nil
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

type MatchInvite struct {
	SenderID string `json:"sender_id"`
	ReceiverID string `json:"receiver_id"`
	Status string `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func (r *Repository) showPendingInvites(ctx context.Context, userID string) ([]MatchInvite, error) {
	rows, err := r.DB.Query(
		ctx,
		`
		SELECT sender_id, receiver_id, status, created_at FROM match_invites
		WHERE status = 'pending' AND receiver_id = $1
		`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invites []MatchInvite
	for rows.Next() {
		var invite MatchInvite
		err := rows.Scan(&invite.SenderID, &invite.ReceiverID, &invite.Status, &invite.CreatedAt)
		if err != nil {
			return nil, err
		}
		invites = append(invites, invite)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return invites, nil
}

func (r *Repository) AcceptInvite(ctx context.Context, sender, receiver string) error {
	result, err := r.DB.Exec(
		ctx,
		`
		UPDATE match_invites SET status = 'accepted' WHERE sender_id = $1 AND receiver_id = $2 AND status = 'pending'
		`,
		sender,
		receiver,
	)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrInviteNotFound
	}

	return nil
}

func (r *Repository) RejectInvite(ctx context.Context, sender, receiver string) error {
	result, err := r.DB.Exec(
		ctx,
		`
		UPDATE match_invites SET status = 'rejected' WHERE sender_id = $1 AND receiver_id = $2 AND status = 'pending'
		`,
		sender,
		receiver,
	)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrInviteNotFound
	}

	return nil
}

