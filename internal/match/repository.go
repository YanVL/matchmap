package match

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB *pgxpool.Pool
}

var ErrInviteConflict = errors.New("invite already exists")
var ErrInviteNotFound = errors.New("invite not found")
var ErrMatchNotFound = errors.New("match not found")

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
	InviteID   string    `json:"invite_id"`
	SenderID   string    `json:"sender_id"`
	ReceiverID string    `json:"receiver_id"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

func (r *Repository) showPendingInvites(ctx context.Context, userID string) ([]MatchInvite, error) {
	rows, err := r.DB.Query(
		ctx,
		`
		SELECT  id, sender_id, receiver_id, status, created_at FROM match_invites
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
		err := rows.Scan(&invite.InviteID, &invite.SenderID, &invite.ReceiverID, &invite.Status, &invite.CreatedAt)
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
		SELECT sender_id, receiver_id 
		FROM match_invites
		WHERE id = $1 AND status = 'pending'
		`,
		inviteID,
	).Scan(&senderID, &receiverID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInviteNotFound
	}
	if err != nil {
		return err
	}

	// Lock the users to prevent race conditions
	_, err = tx.Exec(
		ctx,
		`
		SELECT id FROM users
		WHERE id IN ($1, $2)
		ORDER BY id
		FOR UPDATE
		`,
		senderID,
		receiverID,
	)
	if err != nil {
		return err
	}

	var exists bool
	err = tx.QueryRow(
		ctx,
		`
		SELECT EXISTS (
			SELECT 1 FROM matches
			WHERE (player_1_id = $1 OR player_2_id = $1 OR player_1_id = $2 OR player_2_id = $2)
			AND status = 'active'
		)
		`,
		senderID,
		receiverID,
	).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return ErrUserAlreadyInMatch
	}

	result, err := tx.Exec(
		ctx,
		`
		UPDATE match_invites 
		SET status = 'accepted' 
		WHERE id = $1 AND status = 'pending'
		`,
		inviteID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrInviteNotFound
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

func (r *Repository) FinishMatch(ctx context.Context, matchID, userID string) error {
	result, err := r.DB.Exec(
		ctx,
		`
		UPDATE matches
		SET
			player_1_finished = CASE
				WHEN player_1_id = $2 THEN true
				ELSE player_1_finished
			END,

			player_2_finished = CASE
				WHEN player_2_id = $2 THEN true
				ELSE player_2_finished
			END,

			status = CASE
				WHEN
					(player_1_id = $2 AND player_2_finished)
					OR
					(player_2_id = $2 AND player_1_finished)
				THEN 'finished'
				ELSE status
			END,

			finished_at = CASE
				WHEN
					(player_1_id = $2 AND player_2_finished)
					OR
					(player_2_id = $2 AND player_1_finished)
				THEN NOW()
				ELSE finished_at
			END

		WHERE id = $1
			AND status = 'active'
			AND (player_1_id = $2 OR player_2_id = $2)
		`,
		matchID,
		userID,
	)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrMatchNotFound
	}

	return nil
}

func (r *Repository) RecordMatchResult(ctx context.Context, matchID, userID, matchResult string) error {
	result, err := r.DB.Exec(
		ctx,
		`
		INSERT INTO match_results (match_id, player_id, result)
		VALUES ($1, $2, $3)
		`,
		matchID,
		userID,
		matchResult,
	)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrMatchNotFound
	}

	return nil
}
