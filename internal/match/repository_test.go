package match

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestGetUserMatchStats(t *testing.T) {
	ctx := context.Background()

	db, err := pgxpool.New(
		ctx,
		"postgres://matchmap:matchmap@localhost:5432/matchmap?sslmode=disable",
	)
	if err != nil {
		t.Fatal(err)
	}

	repo := &Repository{
		DB: db,
	}

	service := &Service{
		Repository: repo,
	}

	userID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

	opponent1ID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	opponent2ID := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	opponent3ID := "dddddddd-dddd-dddd-dddd-dddddddddddd"

	match1ID := "11111111-1111-1111-1111-111111111111"
	match2ID := "22222222-2222-2222-2222-222222222222"
	match3ID := "33333333-3333-3333-3333-333333333333"

	cleanup := func() {
		_, err := db.Exec(
			ctx,
			`
			DELETE FROM match_results
			WHERE match_id IN ($1, $2, $3)
			`,
			match1ID,
			match2ID,
			match3ID,
		)
		if err != nil {
			t.Errorf("failed to clean match results: %v", err)
		}

		_, err = db.Exec(
			ctx,
			`
			DELETE FROM matches
			WHERE id IN ($1, $2, $3)
			`,
			match1ID,
			match2ID,
			match3ID,
		)
		if err != nil {
			t.Errorf("failed to clean matches: %v", err)
		}

		_, err = db.Exec(
			ctx,
			`
			DELETE FROM users
			WHERE id IN ($1, $2, $3, $4)
			`,
			userID,
			opponent1ID,
			opponent2ID,
			opponent3ID,
		)
		if err != nil {
			t.Errorf("failed to clean users: %v", err)
		}
	}

	// Remove possible leftovers from a previous interrupted test.
	cleanup()

	t.Cleanup(func() {
		cleanup()
		db.Close()
	})

	// Users
	_, err = db.Exec(
		ctx,
		`
		INSERT INTO users (id, name)
		VALUES
			($1, 'Test User'),
			($2, 'Opponent 1'),
			($3, 'Opponent 2'),
			($4, 'Opponent 3')
		`,
		userID,
		opponent1ID,
		opponent2ID,
		opponent3ID,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Match 1:
	// User: win
	// Opponent: loss
	// Concordant
	_, err = db.Exec(
		ctx,
		`
		INSERT INTO matches (
			id,
			player_1_id,
			player_2_id,
			player_1_finished,
			player_2_finished,
			status
		)
		VALUES ($1, $2, $3, true, true, 'finished')
		`,
		match1ID,
		userID,
		opponent1ID,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(
		ctx,
		`
		INSERT INTO match_results (match_id, player_id, result)
		VALUES
			($1, $2, 'win'),
			($1, $3, 'loss')
		`,
		match1ID,
		userID,
		opponent1ID,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Match 2:
	// User: draw
	// Opponent: draw
	// Concordant
	_, err = db.Exec(
		ctx,
		`
		INSERT INTO matches (
			id,
			player_1_id,
			player_2_id,
			player_1_finished,
			player_2_finished,
			status
		)
		VALUES ($1, $2, $3, true, true, 'finished')
		`,
		match2ID,
		userID,
		opponent2ID,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(
		ctx,
		`
		INSERT INTO match_results (match_id, player_id, result)
		VALUES
			($1, $2, 'draw'),
			($1, $3, 'draw')
		`,
		match2ID,
		userID,
		opponent2ID,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Match 3:
	// User: win
	// Opponent: win
	// Discordant
	_, err = db.Exec(
		ctx,
		`
		INSERT INTO matches (
			id,
			player_1_id,
			player_2_id,
			player_1_finished,
			player_2_finished,
			status
		)
		VALUES ($1, $2, $3, true, true, 'finished')
		`,
		match3ID,
		userID,
		opponent3ID,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(
		ctx,
		`
		INSERT INTO match_results (match_id, player_id, result)
		VALUES
			($1, $2, 'win'),
			($1, $3, 'win')
		`,
		match3ID,
		userID,
		opponent3ID,
	)
	if err != nil {
		t.Fatal(err)
	}

	stats, err := service.GetUserMatchStats(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}

	const tolerance = 0.000001

	if stats.Matches != 3 {
		t.Errorf("expected 3 matches, got %d", stats.Matches)
	}

	if stats.Wins != 2 {
		t.Errorf("expected 2 wins, got %d", stats.Wins)
	}

	if stats.Losses != 0 {
		t.Errorf("expected 0 losses, got %d", stats.Losses)
	}

	if stats.Draws != 1 {
		t.Errorf("expected 1 draw, got %d", stats.Draws)
	}

	if stats.ConcordantMatches != 2 {
		t.Errorf(
			"expected 2 concordant matches, got %d",
			stats.ConcordantMatches,
		)
	}

	if stats.DiscordantMatches != 1 {
		t.Errorf(
			"expected 1 discordant match, got %d",
			stats.DiscordantMatches,
		)
	}

	if stats.ConcordantWins != 1 {
		t.Errorf(
			"expected 1 concordant win, got %d",
			stats.ConcordantWins,
		)
	}

	expectedWinrate := 2.0 / 3.0 * 100

	if math.Abs(stats.Winrate-expectedWinrate) > tolerance {
		t.Errorf(
			"expected winrate %.6f, got %.6f",
			expectedWinrate,
			stats.Winrate,
		)
	}

	expectedConcordanceRate := 2.0 / 3.0 * 100

	if math.Abs(stats.ConcordanceRate-expectedConcordanceRate) > tolerance {
		t.Errorf(
			"expected concordance rate %.6f, got %.6f",
			expectedConcordanceRate,
			stats.ConcordanceRate,
		)
	}

	expectedWinConcordanceRate := 1.0 / 2.0 * 100

	if math.Abs(stats.WinConcordanceRate-expectedWinConcordanceRate) > tolerance {
		t.Errorf(
			"expected win concordance rate %.6f, got %.6f",
			expectedWinConcordanceRate,
			stats.WinConcordanceRate,
		)
	}

	// Match 1: win + concordant = 4
	// Match 2: draw + concordant = 2
	// Match 3: win + discordant = 2
	// Total = 8
	if stats.Score != 8 {
		t.Errorf("expected score 8, got %d", stats.Score)
	}
}

func TestRecordMatchResult(t *testing.T) {
	ctx := context.Background()

	db, err := pgxpool.New(
		ctx,
		"postgres://matchmap:matchmap@localhost:5432/matchmap?sslmode=disable",
	)
	if err != nil {
		t.Fatal(err)
	}

	repo := &Repository{
		DB: db,
	}

	user1ID := "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"
	user2ID := "ffffffff-ffff-ffff-ffff-ffffffffffff"
	matchID := "99999999-9999-9999-9999-999999999999"

	cleanup := func() {
		_, err := db.Exec(
			ctx,
			`DELETE FROM match_results WHERE match_id = $1`,
			matchID,
		)
		if err != nil {
			t.Errorf("failed to clean match results: %v", err)
		}

		_, err = db.Exec(
			ctx,
			`DELETE FROM matches WHERE id = $1`,
			matchID,
		)
		if err != nil {
			t.Errorf("failed to clean match: %v", err)
		}

		_, err = db.Exec(
			ctx,
			`DELETE FROM users WHERE id IN ($1, $2)`,
			user1ID,
			user2ID,
		)
		if err != nil {
			t.Errorf("failed to clean users: %v", err)
		}
	}

	cleanup()

	t.Cleanup(func() {
		cleanup()
		db.Close()
	})

	_, err = db.Exec(
		ctx,
		`
		INSERT INTO users (id, name)
		VALUES
			($1, 'Test User 1'),
			($2, 'Test User 2')
		`,
		user1ID,
		user2ID,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(
		ctx,
		`
		INSERT INTO matches (
			id,
			player_1_id,
			player_2_id,
			player_1_finished,
			player_2_finished,
			status
		)
		VALUES ($1, $2, $3, true, true, 'finished')
		`,
		matchID,
		user1ID,
		user2ID,
	)
	if err != nil {
		t.Fatal(err)
	}

	err = repo.RecordMatchResult(
		ctx,
		matchID,
		user1ID,
		"win",
	)
	if err != nil {
		t.Fatalf(
			"expected result to be recorded successfully, got %v",
			err,
		)
	}

	var result string

	err = db.QueryRow(
		ctx,
		`
		SELECT result
		FROM match_results
		WHERE match_id = $1
			AND player_id = $2
		`,
		matchID,
		user1ID,
	).Scan(&result)
	if err != nil {
		t.Fatal(err)
	}

	if result != "win" {
		t.Fatalf(
			"expected result %q, got %q",
			"win",
			result,
		)
	}
}

func TestRecordMatchResult_AlreadyRecorded(t *testing.T) {
	ctx := context.Background()

	db, err := pgxpool.New(
		ctx,
		"postgres://matchmap:matchmap@localhost:5432/matchmap?sslmode=disable",
	)
	if err != nil {
		t.Fatal(err)
	}

	repo := &Repository{
		DB: db,
	}

	user1ID := "11111111-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	user2ID := "22222222-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	matchID := "33333333-cccc-cccc-cccc-cccccccccccc"

	cleanup := func() {
		_, err := db.Exec(
			ctx,
			`DELETE FROM match_results WHERE match_id = $1`,
			matchID,
		)
		if err != nil {
			t.Errorf("failed to clean match results: %v", err)
		}

		_, err = db.Exec(
			ctx,
			`DELETE FROM matches WHERE id = $1`,
			matchID,
		)
		if err != nil {
			t.Errorf("failed to clean match: %v", err)
		}

		_, err = db.Exec(
			ctx,
			`DELETE FROM users WHERE id IN ($1, $2)`,
			user1ID,
			user2ID,
		)
		if err != nil {
			t.Errorf("failed to clean users: %v", err)
		}
	}

	cleanup()

	t.Cleanup(func() {
		cleanup()
		db.Close()
	})

	_, err = db.Exec(
		ctx,
		`
		INSERT INTO users (id, name)
		VALUES
			($1, 'Test User 1'),
			($2, 'Test User 2')
		`,
		user1ID,
		user2ID,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(
		ctx,
		`
		INSERT INTO matches (
			id,
			player_1_id,
			player_2_id,
			player_1_finished,
			player_2_finished,
			status
		)
		VALUES ($1, $2, $3, true, true, 'finished')
		`,
		matchID,
		user1ID,
		user2ID,
	)
	if err != nil {
		t.Fatal(err)
	}

	err = repo.RecordMatchResult(
		ctx,
		matchID,
		user1ID,
		"win",
	)
	if err != nil {
		t.Fatalf(
			"expected first result to be recorded successfully, got %v",
			err,
		)
	}

	err = repo.RecordMatchResult(
		ctx,
		matchID,
		user1ID,
		"loss",
	)

	if !errors.Is(err, ErrMatchResultAlreadyRecorded) {
		t.Fatalf(
			"expected ErrMatchResultAlreadyRecorded, got %v",
			err,
		)
	}
}

func TestFinishMatch_FirstPlayer(t *testing.T) {
	ctx := context.Background()

	db, err := pgxpool.New(
		ctx,
		"postgres://matchmap:matchmap@localhost:5432/matchmap?sslmode=disable",
	)
	if err != nil {
		t.Fatal(err)
	}

	repo := &Repository{
		DB: db,
	}

	player1ID := "44444444-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	player2ID := "55555555-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	matchID := "66666666-cccc-cccc-cccc-cccccccccccc"

	cleanup := func() {
		_, err := db.Exec(
			ctx,
			`DELETE FROM match_results WHERE match_id = $1`,
			matchID,
		)
		if err != nil {
			t.Errorf("failed to clean match results: %v", err)
		}

		_, err = db.Exec(
			ctx,
			`DELETE FROM matches WHERE id = $1`,
			matchID,
		)
		if err != nil {
			t.Errorf("failed to clean match: %v", err)
		}

		_, err = db.Exec(
			ctx,
			`DELETE FROM users WHERE id IN ($1, $2)`,
			player1ID,
			player2ID,
		)
		if err != nil {
			t.Errorf("failed to clean users: %v", err)
		}
	}

	cleanup()

	t.Cleanup(func() {
		cleanup()
		db.Close()
	})

	_, err = db.Exec(
		ctx,
		`
		INSERT INTO users (id, name)
		VALUES
			($1, 'Player 1'),
			($2, 'Player 2')
		`,
		player1ID,
		player2ID,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(
		ctx,
		`
		INSERT INTO matches (
			id,
			player_1_id,
			player_2_id,
			status
		)
		VALUES ($1, $2, $3, 'active')
		`,
		matchID,
		player1ID,
		player2ID,
	)
	if err != nil {
		t.Fatal(err)
	}

	err = repo.FinishMatch(
		ctx,
		matchID,
		player1ID,
	)
	if err != nil {
		t.Fatalf(
			"expected first player to finish successfully, got %v",
			err,
		)
	}

	var (
		status          string
		player1Finished bool
		player2Finished bool
		finishedAt      *time.Time
	)

	err = db.QueryRow(
		ctx,
		`
		SELECT
			status,
			player_1_finished,
			player_2_finished,
			finished_at
		FROM matches
		WHERE id = $1
		`,
		matchID,
	).Scan(
		&status,
		&player1Finished,
		&player2Finished,
		&finishedAt,
	)
	if err != nil {
		t.Fatal(err)
	}

	if status != "active" {
		t.Errorf(
			"expected status %q, got %q",
			"active",
			status,
		)
	}

	if !player1Finished {
		t.Error("expected player 1 to be marked as finished")
	}

	if player2Finished {
		t.Error("expected player 2 to remain unfinished")
	}

	if finishedAt != nil {
		t.Error("expected finished_at to remain NULL")
	}
}

func TestFinishMatch_SecondPlayer(t *testing.T) {
	ctx := context.Background()

	db, err := pgxpool.New(
		ctx,
		"postgres://matchmap:matchmap@localhost:5432/matchmap?sslmode=disable",
	)
	if err != nil {
		t.Fatal(err)
	}

	repo := &Repository{
		DB: db,
	}

	player1ID := "77777777-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	player2ID := "88888888-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	matchID := "99999999-cccc-cccc-cccc-cccccccccccc"

	cleanup := func() {
		_, err := db.Exec(
			ctx,
			`DELETE FROM match_results WHERE match_id = $1`,
			matchID,
		)
		if err != nil {
			t.Errorf("failed to clean match results: %v", err)
		}

		_, err = db.Exec(
			ctx,
			`DELETE FROM matches WHERE id = $1`,
			matchID,
		)
		if err != nil {
			t.Errorf("failed to clean match: %v", err)
		}

		_, err = db.Exec(
			ctx,
			`DELETE FROM users WHERE id IN ($1, $2)`,
			player1ID,
			player2ID,
		)
		if err != nil {
			t.Errorf("failed to clean users: %v", err)
		}
	}

	cleanup()

	t.Cleanup(func() {
		cleanup()
		db.Close()
	})

	_, err = db.Exec(
		ctx,
		`
		INSERT INTO users (id, name)
		VALUES
			($1, 'Player 1'),
			($2, 'Player 2')
		`,
		player1ID,
		player2ID,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(
		ctx,
		`
		INSERT INTO matches (
			id,
			player_1_id,
			player_2_id,
			player_1_finished,
			player_2_finished,
			status
		)
		VALUES ($1, $2, $3, true, false, 'active')
		`,
		matchID,
		player1ID,
		player2ID,
	)
	if err != nil {
		t.Fatal(err)
	}

	err = repo.FinishMatch(
		ctx,
		matchID,
		player2ID,
	)
	if err != nil {
		t.Fatalf(
			"expected second player to finish successfully, got %v",
			err,
		)
	}

	var (
		status          string
		player1Finished bool
		player2Finished bool
		finishedAt      *time.Time
	)

	err = db.QueryRow(
		ctx,
		`
		SELECT
			status,
			player_1_finished,
			player_2_finished,
			finished_at
		FROM matches
		WHERE id = $1
		`,
		matchID,
	).Scan(
		&status,
		&player1Finished,
		&player2Finished,
		&finishedAt,
	)
	if err != nil {
		t.Fatal(err)
	}

	if status != "finished" {
		t.Errorf(
			"expected status %q, got %q",
			"finished",
			status,
		)
	}

	if !player1Finished {
		t.Error("expected player 1 to remain finished")
	}

	if !player2Finished {
		t.Error("expected player 2 to be marked as finished")
	}

	if finishedAt == nil {
		t.Error("expected finished_at to be set")
	}
}

func TestFinishMatch_UserNotInMatch(t *testing.T) {
	ctx := context.Background()

	db, err := pgxpool.New(
		ctx,
		"postgres://matchmap:matchmap@localhost:5432/matchmap?sslmode=disable",
	)
	if err != nil {
		t.Fatal(err)
	}

	repo := &Repository{
		DB: db,
	}

	player1ID := "aaaaaaaa-bbbb-cccc-dddd-aaaaaaaaaaaa"
	player2ID := "bbbbbbbb-cccc-dddd-eeee-bbbbbbbbbbbb"
	outsiderID := "cccccccc-dddd-eeee-ffff-cccccccccccc"
	matchID := "dddddddd-eeee-ffff-aaaa-dddddddddddd"

	cleanup := func() {
		_, err := db.Exec(
			ctx,
			`DELETE FROM match_results WHERE match_id = $1`,
			matchID,
		)
		if err != nil {
			t.Errorf("failed to clean match results: %v", err)
		}

		_, err = db.Exec(
			ctx,
			`DELETE FROM matches WHERE id = $1`,
			matchID,
		)
		if err != nil {
			t.Errorf("failed to clean match: %v", err)
		}

		_, err = db.Exec(
			ctx,
			`DELETE FROM users WHERE id IN ($1, $2, $3)`,
			player1ID,
			player2ID,
			outsiderID,
		)
		if err != nil {
			t.Errorf("failed to clean users: %v", err)
		}
	}

	cleanup()

	t.Cleanup(func() {
		cleanup()
		db.Close()
	})

	_, err = db.Exec(
		ctx,
		`
		INSERT INTO users (id, name)
		VALUES
			($1, 'Player 1'),
			($2, 'Player 2'),
			($3, 'Outsider')
		`,
		player1ID,
		player2ID,
		outsiderID,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(
		ctx,
		`
		INSERT INTO matches (
			id,
			player_1_id,
			player_2_id,
			status
		)
		VALUES ($1, $2, $3, 'active')
		`,
		matchID,
		player1ID,
		player2ID,
	)
	if err != nil {
		t.Fatal(err)
	}

	err = repo.FinishMatch(
		ctx,
		matchID,
		outsiderID,
	)

	if !errors.Is(err, ErrMatchNotFound) {
		t.Fatalf(
			"expected ErrMatchNotFound, got %v",
			err,
		)
	}
}

func TestFinishMatch_AlreadyFinished(t *testing.T) {
	ctx := context.Background()

	db, err := pgxpool.New(
		ctx,
		"postgres://matchmap:matchmap@localhost:5432/matchmap?sslmode=disable",
	)
	if err != nil {
		t.Fatal(err)
	}

	repo := &Repository{
		DB: db,
	}

	player1ID := "eeeeeeee-ffff-aaaa-bbbb-eeeeeeeeeeee"
	player2ID := "ffffffff-aaaa-bbbb-cccc-ffffffffffff"
	matchID := "aaaaaaaa-bbbb-cccc-dddd-aaaaaaaaaaaa"

	cleanup := func() {
		_, err := db.Exec(
			ctx,
			`DELETE FROM match_results WHERE match_id = $1`,
			matchID,
		)
		if err != nil {
			t.Errorf("failed to clean match results: %v", err)
		}

		_, err = db.Exec(
			ctx,
			`DELETE FROM matches WHERE id = $1`,
			matchID,
		)
		if err != nil {
			t.Errorf("failed to clean match: %v", err)
		}

		_, err = db.Exec(
			ctx,
			`DELETE FROM users WHERE id IN ($1, $2)`,
			player1ID,
			player2ID,
		)
		if err != nil {
			t.Errorf("failed to clean users: %v", err)
		}
	}

	cleanup()

	t.Cleanup(func() {
		cleanup()
		db.Close()
	})

	_, err = db.Exec(
		ctx,
		`
		INSERT INTO users (id, name)
		VALUES
			($1, 'Player 1'),
			($2, 'Player 2')
		`,
		player1ID,
		player2ID,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(
		ctx,
		`
		INSERT INTO matches (
			id,
			player_1_id,
			player_2_id,
			player_1_finished,
			player_2_finished,
			status,
			finished_at
		)
		VALUES (
			$1,
			$2,
			$3,
			true,
			true,
			'finished',
			NOW()
		)
		`,
		matchID,
		player1ID,
		player2ID,
	)
	if err != nil {
		t.Fatal(err)
	}

	err = repo.FinishMatch(
		ctx,
		matchID,
		player1ID,
	)

	if !errors.Is(err, ErrMatchNotFound) {
		t.Fatalf(
			"expected ErrMatchNotFound, got %v",
			err,
		)
	}
}

func TestAcceptInviteAndCreateMatch_Concurrent(t *testing.T) {
	ctx := context.Background()

	db, err := pgxpool.New(
		ctx,
		"postgres://matchmap:matchmap@localhost:5432/matchmap?sslmode=disable",
	)
	if err != nil {
		t.Fatal(err)
	}

	repo := &Repository{
		DB: db,
	}

	aliceID := "11111111-aaaa-bbbb-cccc-111111111111"
	bobID := "22222222-aaaa-bbbb-cccc-222222222222"
	carlosID := "33333333-aaaa-bbbb-cccc-333333333333"

	inviteAB := "aaaaaaaa-aaaa-bbbb-cccc-aaaaaaaaaaaa"
	inviteAC := "bbbbbbbb-bbbb-cccc-dddd-bbbbbbbbbbbb"

	cleanup := func() {
		_, err := db.Exec(
			ctx,
			`
			DELETE FROM matches
			WHERE player_1_id IN ($1, $2, $3)
			   OR player_2_id IN ($1, $2, $3)
			`,
			aliceID,
			bobID,
			carlosID,
		)
		if err != nil {
			t.Errorf("failed to clean matches: %v", err)
		}

		_, err = db.Exec(
			ctx,
			`
			DELETE FROM match_invites
			WHERE id IN ($1, $2)
			`,
			inviteAB,
			inviteAC,
		)
		if err != nil {
			t.Errorf("failed to clean invites: %v", err)
		}

		_, err = db.Exec(
			ctx,
			`
			DELETE FROM users
			WHERE id IN ($1, $2, $3)
			`,
			aliceID,
			bobID,
			carlosID,
		)
		if err != nil {
			t.Errorf("failed to clean users: %v", err)
		}
	}

	cleanup()

	t.Cleanup(func() {
		cleanup()
		db.Close()
	})

	_, err = db.Exec(
		ctx,
		`
		INSERT INTO users (id, name)
		VALUES
			($1, 'Alice'),
			($2, 'Bob'),
			($3, 'Carlos')
		`,
		aliceID,
		bobID,
		carlosID,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(
		ctx,
		`
    INSERT INTO match_invites (
        id,
        sender_id,
        receiver_id,
        status
    )
    VALUES
        ($1, $2, $3, 'pending'),
        ($4, $5, $6, 'pending')
    `,
		inviteAB,
		aliceID,
		bobID,
		inviteAC,
		aliceID,
		carlosID,
	)
	if err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	errCh := make(chan error, 2)

	go func() {
		<-start
		errCh <- repo.AcceptInviteAndCreateMatch(
			ctx,
			inviteAB,
		)
	}()

	go func() {
		<-start
		errCh <- repo.AcceptInviteAndCreateMatch(
			ctx,
			inviteAC,
		)
	}()

	close(start)

	err1 := <-errCh
	err2 := <-errCh

	successes := 0
	alreadyInMatch := 0

	for _, err := range []error{err1, err2} {
		switch {
		case err == nil:
			successes++

		case errors.Is(err, ErrUserAlreadyInMatch):
			alreadyInMatch++

		default:
			t.Fatalf(
				"unexpected error: %v",
				err,
			)
		}
	}

	if successes != 1 {
		t.Fatalf(
			"expected exactly 1 successful match creation, got %d",
			successes,
		)
	}

	if alreadyInMatch != 1 {
		t.Fatalf(
			"expected exactly 1 ErrUserAlreadyInMatch, got %d",
			alreadyInMatch,
		)
	}

	var matchCount int

	err = db.QueryRow(
		ctx,
		`
		SELECT COUNT(*)
		FROM matches
		WHERE status = 'active'
		  AND (
		      player_1_id IN ($1, $2, $3)
		      OR player_2_id IN ($1, $2, $3)
		  )
		`,
		aliceID,
		bobID,
		carlosID,
	).Scan(&matchCount)

	if err != nil {
		t.Fatal(err)
	}

	if matchCount != 1 {
		t.Fatalf(
			"expected exactly 1 active match, got %d",
			matchCount,
		)
	}
}
