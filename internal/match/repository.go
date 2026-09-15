package match

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5"
)

type Repository struct {
	DB *pgxpool.Pool
}

var ErrInviteConflict = errors.New("invite already exists")
var ErrInviteNotFound = errors.New("invite not found")

func (r *Repository) CreateInvite(ctx context.Context, sender, receiver string) (string, error) {
	var inviteID string
	err := r.DB.QueryRow(
		ctx,
		`
		INSERT INTO match_invites (sender_id, receiver_id, status, created_at)
		VALUES ($1, $2, 'pending', NOW())
		RETURNING id
		`,
		sender,
		receiver,
	).Scan(&inviteID)

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) &&
			pgErr.Code == "23505" &&
			pgErr.ConstraintName == "match_invites_pending_pair_idx" {
			return "", ErrInviteConflict
		}

		return "", err
	}

	return inviteID, nil
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
	SenderID   string    `json:"sender_id"`
	ReceiverID string    `json:"receiver_id"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
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

func (r *Repository) AcceptInviteAndCreateMatch(ctx context.Context, inviteID string) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var senderID, receiverID string

	err = tx.QueryRow(
		ctx,
		`
		UPDATE match_invites 
		SET status = 'accepted' 
		WHERE id = $1 AND status = 'pending'
		RETURNING sender_id, receiver_id
		`,
		inviteID,
	).Scan(&senderID, &receiverID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInviteNotFound
		}
		return err
	}

	_, err = tx.Exec(
		ctx,
		`
		INSERT INTO matches (player_1_id, player_2_id, created_at)
		VALUES ($1, $2, NOW())
		`,
		senderID,
		receiverID,
	)

	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) RejectInvite(ctx context.Context, inviteID string) error {
	result, err := r.DB.Exec(
		ctx,
		`
		UPDATE match_invites SET status = 'rejected' WHERE id = $1 AND status = 'pending'
		`,
		inviteID,
	)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrInviteNotFound
	}

	return nil
}

func (r *Repository) GetInviteByID(ctx context.Context, inviteID string) (*MatchInvite, error) {
	var invite MatchInvite
	err := r.DB.QueryRow(
		ctx,
		`
		SELECT sender_id, receiver_id, status, created_at FROM match_invites
		WHERE id = $1
		`,
		inviteID,
	).Scan(&invite.SenderID, &invite.ReceiverID, &invite.Status, &invite.CreatedAt)

	if err != nil {
		return nil, err
	}

	return &invite, nil
}

func (r *Repository) IsUserInMatch(ctx context.Context, userID string) (bool, error) {
	var exists bool
	err := r.DB.QueryRow(
		ctx,
		`
		SELECT EXISTS (
			SELECT 1 FROM matches
			WHERE (player_1_id = $1 OR player_2_id = $1) AND status = 'active'
		)
		`,
		userID,
	).Scan(&exists)

	return exists, err
}