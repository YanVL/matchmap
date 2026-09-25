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

func (r *Repository) GetPendingInvites(ctx context.Context, userID string) ([]MatchInvite, error) {
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

func (r *Repository) AcceptInviteAndCreateMatch(ctx context.Context, inviteID string) (matchID, senderID, receiverID string, err error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return "", "", "", err
	}
	defer tx.Rollback(ctx)

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
		return "", "", "", ErrInviteNotFound
	}
	if err != nil {
		return "", "", "", err
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
		return "", "", "", err
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
		return "", "", "", err
	}
	if exists {
		return "", "", "", ErrUserAlreadyInMatch
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
		return "", "", "", err
	}
	if result.RowsAffected() == 0 {
		return "", "", "", ErrInviteNotFound
	}

	err = tx.QueryRow(
		ctx,
		`
		INSERT INTO matches (player_1_id, player_2_id, created_at)
		VALUES ($1, $2, NOW())
		RETURNING id
		`,
		senderID,
		receiverID,
	).Scan(&matchID)

	if err != nil {
		return "", "", "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", "", "", err
	}

	return matchID, senderID, receiverID, nil
}

func (r *Repository) RejectInvite(ctx context.Context, inviteID string) (senderID string, err error) {
	err = r.DB.QueryRow(
		ctx,
		`
		UPDATE match_invites 
		SET status = 'rejected' 
		WHERE id = $1 AND status = 'pending'
		RETURNING sender_id
		`,
		inviteID,
	).Scan(&senderID)

	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrInviteNotFound
	}

	if err != nil {
		return "", err
	}

	return senderID, nil
}

func (r *Repository) FinishMatch(ctx context.Context, matchID, userID string) (player2ID string, bothFinished bool, err error) {
	err = r.DB.QueryRow(
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
		RETURNING 
			CASE
				WHEN player_1_id = $2 THEN player_2_id
				ELSE player_1_id
			END,
			player_1_finished AND player_2_finished
		`,
		matchID,
		userID,
	).Scan(&player2ID, &bothFinished)

	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, ErrMatchNotFound
	}

	if err != nil {
		return "", false, err
	}

	return player2ID, bothFinished, nil
}

func (r *Repository) RecordMatchResult(ctx context.Context, matchID, userID, matchResult string) (otherPlayerID string, err error) {
    err = r.DB.QueryRow(
        ctx,
        `
        INSERT INTO match_results (match_id, player_id, result)
        VALUES ($1, $2, $3)
        RETURNING (
            SELECT CASE
                WHEN player_1_id = $2 THEN player_2_id
                ELSE player_1_id
            END
            FROM matches
            WHERE id = $1
        )
        `,
        matchID,
        userID,
        matchResult,
    ).Scan(&otherPlayerID)

    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return "", ErrMatchNotFound
        }

        if isUniqueViolation(err) {
            return "", ErrMatchResultAlreadyRecorded
        }

        return "", err
    }

    return otherPlayerID, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return true
	}
	return false
}

type Match struct {
	ID              string     `json:"id"`
	Player1ID       string     `json:"player_1_id"`
	Player2ID       string     `json:"player_2_id"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
	Player1Finished bool       `json:"player_1_finished"`
	Player2Finished bool       `json:"player_2_finished"`
}

func (r *Repository) GetMatchByID(ctx context.Context, matchID string) (*Match, error) {
	var match Match
	err := r.DB.QueryRow(
		ctx,
		`
		SELECT id, player_1_id, player_2_id, status, created_at, finished_at, player_1_finished, player_2_finished
		FROM matches
		WHERE id = $1
		`,
		matchID,
	).Scan(&match.ID, &match.Player1ID, &match.Player2ID, &match.Status, &match.CreatedAt, &match.FinishedAt, &match.Player1Finished, &match.Player2Finished)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMatchNotFound
	}
	if err != nil {
		return nil, err
	}

	return &match, nil
}

type UserMatchResult struct {
	MatchID        string
	UserResult     string
	OpponentResult string
}

func (r *Repository) GetUserMatchResults(ctx context.Context, userID string) ([]UserMatchResult, error) {
	rows, err := r.DB.Query(
		ctx,
		`
        SELECT
            mr.match_id,
            mr.result,
            opponent.result
        FROM match_results mr
        JOIN match_results opponent
            ON opponent.match_id = mr.match_id
            AND opponent.player_id <> mr.player_id
        WHERE mr.player_id = $1
        `,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []UserMatchResult

	for rows.Next() {
		var result UserMatchResult

		if err := rows.Scan(
			&result.MatchID,
			&result.UserResult,
			&result.OpponentResult,
		); err != nil {
			return nil, err
		}

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
