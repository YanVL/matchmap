package match

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestIsConcordant(t *testing.T) {
	tests := []struct {
		name     string
		result1  string
		result2  string
		expected bool
	}{
		{
			name:     "win and loss",
			result1:  "win",
			result2:  "loss",
			expected: true,
		},
		{
			name:     "loss and win",
			result1:  "loss",
			result2:  "win",
			expected: true,
		},
		{
			name:     "draw and draw",
			result1:  "draw",
			result2:  "draw",
			expected: true,
		},
		{
			name:     "win and win",
			result1:  "win",
			result2:  "win",
			expected: false,
		},
		{
			name:     "loss and loss",
			result1:  "loss",
			result2:  "loss",
			expected: false,
		},
		{
			name:     "win and draw",
			result1:  "win",
			result2:  "draw",
			expected: false,
		},
		{
			name:     "draw and loss",
			result1:  "draw",
			result2:  "loss",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := isConcordant(test.result1, test.result2)

			if result != test.expected {
				t.Errorf(
					"expected %v, got %v",
					test.expected,
					result,
				)
			}
		})
	}
}

func TestCalculateMatchScore(t *testing.T) {
	tests := []struct {
		name        string
		result      string
		concordance string
		expected    int
	}{
		{
			name:        "win and concordant",
			result:      "win",
			concordance: "concordant",
			expected:    4,
		},
		{
			name:        "win and discordant",
			result:      "win",
			concordance: "discordant",
			expected:    2,
		},
		{
			name:        "draw and concordant",
			result:      "draw",
			concordance: "concordant",
			expected:    2,
		},
		{
			name:        "draw and discordant",
			result:      "draw",
			concordance: "discordant",
			expected:    0,
		},
		{
			name:        "loss and concordant",
			result:      "loss",
			concordance: "concordant",
			expected:    1,
		},
		{
			name:        "loss and discordant",
			result:      "loss",
			concordance: "discordant",
			expected:    -1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := calculateMatchScore(
				test.result,
				test.concordance,
			)

			if result != test.expected {
				t.Errorf(
					"expected %d, got %d",
					test.expected,
					result,
				)
			}
		})
	}
}

func TestCalculateMatchStats(t *testing.T) {
	const tolerance = 0.000001

	tests := []struct {
		name                       string
		results                    []UserMatchResult
		expectedMatches            int
		expectedWins               int
		expectedLosses             int
		expectedDraws              int
		expectedConcordant         int
		expectedDiscordant         int
		expectedScore              int
		expectedWinrate            float64
		expectedConcordanceRate    float64
		expectedWinConcordanceRate float64
	}{
		{
			name: "all concordant",
			results: []UserMatchResult{
				{
					UserResult:     "win",
					OpponentResult: "loss",
				},
				{
					UserResult:     "loss",
					OpponentResult: "win",
				},
				{
					UserResult:     "draw",
					OpponentResult: "draw",
				},
			},
			expectedMatches:            3,
			expectedWins:               1,
			expectedLosses:             1,
			expectedDraws:              1,
			expectedConcordant:         3,
			expectedDiscordant:         0,
			expectedScore:              7,
			expectedWinrate:            33.33333333333333,
			expectedConcordanceRate:    100.0,
			expectedWinConcordanceRate: 100.0,
		},
		{
			name: "one discordant match",
			results: []UserMatchResult{
				{
					UserResult:     "win",
					OpponentResult: "loss",
				},
				{
					UserResult:     "loss",
					OpponentResult: "win",
				},
				{
					UserResult:     "draw",
					OpponentResult: "win",
				},
			},
			expectedMatches:            3,
			expectedWins:               1,
			expectedLosses:             1,
			expectedDraws:              1,
			expectedConcordant:         2,
			expectedDiscordant:         1,
			expectedScore:              5,
			expectedWinrate:            33.33333333333333,
			expectedConcordanceRate:    66.66666666666667,
			expectedWinConcordanceRate: 100.0,
		},
		{
			name: "all discordant",
			results: []UserMatchResult{
				{
					UserResult:     "win",
					OpponentResult: "win",
				},
				{
					UserResult:     "loss",
					OpponentResult: "loss",
				},
				{
					UserResult:     "draw",
					OpponentResult: "win",
				},
			},
			expectedMatches:            3,
			expectedWins:               1,
			expectedLosses:             1,
			expectedDraws:              1,
			expectedConcordant:         0,
			expectedDiscordant:         3,
			expectedScore:              1,
			expectedWinrate:            33.33333333333333,
			expectedConcordanceRate:    0.0,
			expectedWinConcordanceRate: 0.0,
		},
		{
			name:                       "no matches",
			results:                    []UserMatchResult{},
			expectedMatches:            0,
			expectedWins:               0,
			expectedLosses:             0,
			expectedDraws:              0,
			expectedConcordant:         0,
			expectedDiscordant:         0,
			expectedScore:              0,
			expectedWinrate:            0.0,
			expectedConcordanceRate:    0.0,
			expectedWinConcordanceRate: 0.0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stats := calculateMatchStats(test.results)

			if stats.Matches != test.expectedMatches {
				t.Errorf(
					"expected %d matches, got %d",
					test.expectedMatches,
					stats.Matches,
				)
			}

			if stats.Wins != test.expectedWins {
				t.Errorf(
					"expected %d wins, got %d",
					test.expectedWins,
					stats.Wins,
				)
			}

			if stats.Losses != test.expectedLosses {
				t.Errorf(
					"expected %d losses, got %d",
					test.expectedLosses,
					stats.Losses,
				)
			}

			if stats.Draws != test.expectedDraws {
				t.Errorf(
					"expected %d draws, got %d",
					test.expectedDraws,
					stats.Draws,
				)
			}

			if stats.ConcordantMatches != test.expectedConcordant {
				t.Errorf(
					"expected %d concordant matches, got %d",
					test.expectedConcordant,
					stats.ConcordantMatches,
				)
			}

			if stats.DiscordantMatches != test.expectedDiscordant {
				t.Errorf(
					"expected %d discordant matches, got %d",
					test.expectedDiscordant,
					stats.DiscordantMatches,
				)
			}

			if stats.Score != test.expectedScore {
				t.Errorf(
					"expected score %d, got %d",
					test.expectedScore,
					stats.Score,
				)
			}

			if math.Abs(stats.Winrate-test.expectedWinrate) > tolerance {
				t.Errorf(
					"expected winrate %f, got %f",
					test.expectedWinrate,
					stats.Winrate,
				)
			}

			if math.Abs(stats.ConcordanceRate-test.expectedConcordanceRate) > tolerance {
				t.Errorf(
					"expected concordance rate %f, got %f",
					test.expectedConcordanceRate,
					stats.ConcordanceRate,
				)
			}

			if math.Abs(stats.WinConcordanceRate-test.expectedWinConcordanceRate) > tolerance {
				t.Errorf(
					"expected win concordance rate %f, got %f",
					test.expectedWinConcordanceRate,
					stats.WinConcordanceRate,
				)
			}
		})
	}
}

func TestRecordMatchResult_InvalidResult(t *testing.T) {
	service := &Service{}

	err := service.RecordMatchResult(
		context.Background(),
		"match-id",
		"user-id",
		"banana",
	)

	if !errors.Is(err, ErrInvalidResult) {
		t.Fatalf(
			"expected ErrInvalidResult, got %v",
			err,
		)
	}
}

func TestRecordMatchResult_UserNotInMatch(t *testing.T) {
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

	matchID := "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"
	player1ID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	player2ID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	outsiderID := "cccccccc-cccc-cccc-cccc-cccccccccccc"

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
			player_1_finished,
			player_2_finished,
			status
		)
		VALUES ($1, $2, $3, true, true, 'finished')
		`,
		matchID,
		player1ID,
		player2ID,
	)
	if err != nil {
		t.Fatal(err)
	}

	err = service.RecordMatchResult(
		ctx,
		matchID,
		outsiderID,
		"win",
	)

	if !errors.Is(err, ErrMatchNotFound) {
		t.Fatalf(
			"expected ErrMatchNotFound, got %v",
			err,
		)
	}
}

func TestRecordMatchResult_MatchNotFinished(t *testing.T) {
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

	matchID := "ffffffff-ffff-ffff-ffff-ffffffffffff"
	player1ID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	player2ID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"

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

	err = service.RecordMatchResult(
		ctx,
		matchID,
		player1ID,
		"win",
	)

	if !errors.Is(err, ErrMatchNotFinished) {
		t.Fatalf(
			"expected ErrMatchNotFinished, got %v",
			err,
		)
	}
}

func TestGetMatchByID(t *testing.T) {
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

	player1ID := "11111111-aaaa-bbbb-cccc-111111111111"
	player2ID := "22222222-bbbb-cccc-dddd-222222222222"
	matchID := "33333333-cccc-dddd-eeee-333333333333"

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

	match, err := repo.GetMatchByID(ctx, matchID)
	if err != nil {
		t.Fatal(err)
	}

	if match.ID != matchID {
		t.Errorf(
			"expected ID %q, got %q",
			matchID,
			match.ID,
		)
	}

	if match.Player1ID != player1ID {
		t.Errorf(
			"expected player 1 ID %q, got %q",
			player1ID,
			match.Player1ID,
		)
	}

	if match.Player2ID != player2ID {
		t.Errorf(
			"expected player 2 ID %q, got %q",
			player2ID,
			match.Player2ID,
		)
	}

	if match.Status != "finished" {
		t.Errorf(
			"expected status %q, got %q",
			"finished",
			match.Status,
		)
	}

	if !match.Player1Finished {
		t.Error("expected player 1 to be finished")
	}

	if !match.Player2Finished {
		t.Error("expected player 2 to be finished")
	}

	if match.FinishedAt == nil {
		t.Error("expected finished_at to be set")
	}
}

func TestGetMatchByID_NotFound(t *testing.T) {
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

	defer db.Close()

	matchID := "ffffffff-eeee-dddd-cccc-bbbbbbbbbbbb"

	_, err = repo.GetMatchByID(ctx, matchID)

	if !errors.Is(err, ErrMatchNotFound) {
		t.Fatalf(
			"expected ErrMatchNotFound, got %v",
			err,
		)
	}
}

func TestAcceptInviteAndCreateMatch(t *testing.T) {
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

	senderID := "11111111-2222-3333-4444-111111111111"
	receiverID := "22222222-3333-4444-5555-222222222222"
	inviteID := "33333333-4444-5555-6666-333333333333"

	cleanup := func() {
		_, err := db.Exec(
			ctx,
			`
			DELETE FROM match_results
			WHERE match_id IN (
				SELECT id
				FROM matches
				WHERE player_1_id IN ($1, $2)
				   OR player_2_id IN ($1, $2)
			)
			`,
			senderID,
			receiverID,
		)
		if err != nil {
			t.Errorf("failed to clean match results: %v", err)
		}

		_, err = db.Exec(
			ctx,
			`
			DELETE FROM matches
			WHERE player_1_id IN ($1, $2)
			   OR player_2_id IN ($1, $2)
			`,
			senderID,
			receiverID,
		)
		if err != nil {
			t.Errorf("failed to clean matches: %v", err)
		}

		_, err = db.Exec(
			ctx,
			`DELETE FROM match_invites WHERE id = $1`,
			inviteID,
		)
		if err != nil {
			t.Errorf("failed to clean invite: %v", err)
		}

		_, err = db.Exec(
			ctx,
			`DELETE FROM users WHERE id IN ($1, $2)`,
			senderID,
			receiverID,
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
			($1, 'Sender'),
			($2, 'Receiver')
		`,
		senderID,
		receiverID,
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
		VALUES ($1, $2, $3, 'pending')
		`,
		inviteID,
		senderID,
		receiverID,
	)
	if err != nil {
		t.Fatal(err)
	}

	err = repo.AcceptInviteAndCreateMatch(
		ctx,
		inviteID,
	)
	if err != nil {
		t.Fatalf(
			"expected invite to be accepted successfully, got %v",
			err,
		)
	}

	var inviteStatus string

	err = db.QueryRow(
		ctx,
		`
		SELECT status
		FROM match_invites
		WHERE id = $1
		`,
		inviteID,
	).Scan(&inviteStatus)
	if err != nil {
		t.Fatal(err)
	}

	if inviteStatus != "accepted" {
		t.Errorf(
			"expected invite status %q, got %q",
			"accepted",
			inviteStatus,
		)
	}

	var (
		matchID string
		player1 string
		player2 string
		status  string
	)

	err = db.QueryRow(
		ctx,
		`
		SELECT id, player_1_id, player_2_id, status
		FROM matches
		WHERE player_1_id = $1
		  AND player_2_id = $2
		`,
		senderID,
		receiverID,
	).Scan(
		&matchID,
		&player1,
		&player2,
		&status,
	)
	if err != nil {
		t.Fatal(err)
	}

	if matchID == "" {
		t.Error("expected match ID to be created")
	}

	if player1 != senderID {
		t.Errorf(
			"expected player 1 %q, got %q",
			senderID,
			player1,
		)
	}

	if player2 != receiverID {
		t.Errorf(
			"expected player 2 %q, got %q",
			receiverID,
			player2,
		)
	}

	if status != "active" {
		t.Errorf(
			"expected match status %q, got %q",
			"active",
			status,
		)
	}
}

func TestAcceptInviteAndCreateMatch_UserAlreadyInMatch(t *testing.T) {
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

	senderID := "44444444-2222-3333-4444-444444444444"
	receiverID := "55555555-3333-4444-5555-555555555555"
	existingPlayerID := "66666666-4444-5555-6666-666666666666"

	inviteID := "77777777-5555-6666-7777-777777777777"
	existingMatchID := "88888888-6666-7777-8888-888888888888"

	cleanup := func() {
		_, err := db.Exec(
			ctx,
			`DELETE FROM match_results WHERE match_id IN ($1, $2)`,
			existingMatchID,
			"99999999-7777-8888-9999-999999999999",
		)
		if err != nil {
			t.Errorf("failed to clean match results: %v", err)
		}

		_, err = db.Exec(
			ctx,
			`
			DELETE FROM matches
			WHERE id IN ($1, $2)
			`,
			existingMatchID,
			"99999999-7777-8888-9999-999999999999",
		)
		if err != nil {
			t.Errorf("failed to clean matches: %v", err)
		}

		_, err = db.Exec(
			ctx,
			`DELETE FROM match_invites WHERE id = $1`,
			inviteID,
		)
		if err != nil {
			t.Errorf("failed to clean invite: %v", err)
		}

		_, err = db.Exec(
			ctx,
			`
			DELETE FROM users
			WHERE id IN ($1, $2, $3)
			`,
			senderID,
			receiverID,
			existingPlayerID,
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
			($1, 'Sender'),
			($2, 'Receiver'),
			($3, 'Existing Player')
		`,
		senderID,
		receiverID,
		existingPlayerID,
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
		VALUES ($1, $2, $3, 'pending')
		`,
		inviteID,
		senderID,
		receiverID,
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
		existingMatchID,
		senderID,
		existingPlayerID,
	)
	if err != nil {
		t.Fatal(err)
	}

	err = repo.AcceptInviteAndCreateMatch(
		ctx,
		inviteID,
	)

	if !errors.Is(err, ErrUserAlreadyInMatch) {
		t.Fatalf(
			"expected ErrUserAlreadyInMatch, got %v",
			err,
		)
	}

	var inviteStatus string

	err = db.QueryRow(
		ctx,
		`
		SELECT status
		FROM match_invites
		WHERE id = $1
		`,
		inviteID,
	).Scan(&inviteStatus)
	if err != nil {
		t.Fatal(err)
	}

	if inviteStatus != "pending" {
		t.Errorf(
			"expected invite status %q after rollback, got %q",
			"pending",
			inviteStatus,
		)
	}

	var matchCount int

	err = db.QueryRow(
		ctx,
		`
		SELECT COUNT(*)
		FROM matches
		WHERE player_1_id = $1
		  AND player_2_id = $2
		`,
		senderID,
		receiverID,
	).Scan(&matchCount)
	if err != nil {
		t.Fatal(err)
	}

	if matchCount != 0 {
		t.Errorf(
			"expected no new match to be created, got %d",
			matchCount,
		)
	}
}

func TestAcceptInviteAndCreateMatch_InviteNotFound(t *testing.T) {
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

	inviteID := "aaaaaaaa-5555-6666-7777-aaaaaaaaaaaa"

	t.Cleanup(func() {
		db.Close()
	})

	err = repo.AcceptInviteAndCreateMatch(
		ctx,
		inviteID,
	)

	if !errors.Is(err, ErrInviteNotFound) {
		t.Fatalf(
			"expected ErrInviteNotFound, got %v",
			err,
		)
	}
}

func TestRejectInvite(t *testing.T) {
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

	senderID := "11111111-2222-3333-4444-111111111111"
	receiverID := "22222222-3333-4444-5555-222222222222"
	inviteID := "33333333-4444-5555-6666-333333333333"

	cleanup := func() {
		_, err := db.Exec(
			ctx,
			`DELETE FROM match_invites WHERE id = $1`,
			inviteID,
		)
		if err != nil {
			t.Errorf("failed to clean invite: %v", err)
		}

		_, err = db.Exec(
			ctx,
			`DELETE FROM users WHERE id IN ($1, $2)`,
			senderID,
			receiverID,
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
			($1, 'Sender'),
			($2, 'Receiver')
		`,
		senderID,
		receiverID,
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
		VALUES ($1, $2, $3, 'pending')
		`,
		inviteID,
		senderID,
		receiverID,
	)
	if err != nil {
		t.Fatal(err)
	}

	err = repo.RejectInvite(ctx, inviteID)
	if err != nil {
		t.Fatalf(
			"expected invite to be rejected successfully, got %v",
			err,
		)
	}

	var status string

	err = db.QueryRow(
		ctx,
		`
		SELECT status
		FROM match_invites
		WHERE id = $1
		`,
		inviteID,
	).Scan(&status)
	if err != nil {
		t.Fatal(err)
	}

	if status != "rejected" {
		t.Errorf(
			"expected invite status %q, got %q",
			"rejected",
			status,
		)
	}
}

func TestRejectInvite_InviteNotFound(t *testing.T) {
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

	t.Cleanup(func() {
		db.Close()
	})

	inviteID := "aaaaaaaa-1111-2222-3333-aaaaaaaaaaaa"

	err = repo.RejectInvite(ctx, inviteID)

	if !errors.Is(err, ErrInviteNotFound) {
		t.Fatalf(
			"expected ErrInviteNotFound, got %v",
			err,
		)
	}
}

func TestCreateInvite(t *testing.T) {
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

	senderID := "11111111-2222-3333-4444-111111111111"
	receiverID := "22222222-3333-4444-5555-222222222222"

	cleanup := func() {
		_, err := db.Exec(
			ctx,
			`
			DELETE FROM match_invites
			WHERE sender_id IN ($1, $2)
			   OR receiver_id IN ($1, $2)
			`,
			senderID,
			receiverID,
		)
		if err != nil {
			t.Errorf("failed to clean invites: %v", err)
		}

		_, err = db.Exec(
			ctx,
			`DELETE FROM users WHERE id IN ($1, $2)`,
			senderID,
			receiverID,
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
			($1, 'Sender'),
			($2, 'Receiver')
		`,
		senderID,
		receiverID,
	)
	if err != nil {
		t.Fatal(err)
	}

	inviteID, err := repo.CreateInvite(
		ctx,
		senderID,
		receiverID,
	)
	if err != nil {
		t.Fatalf(
			"expected invite to be created successfully, got %v",
			err,
		)
	}

	if inviteID == "" {
		t.Error("expected invite ID to be returned")
	}

	var (
		dbSenderID   string
		dbReceiverID string
		status       string
	)

	err = db.QueryRow(
		ctx,
		`
		SELECT sender_id, receiver_id, status
		FROM match_invites
		WHERE id = $1
		`,
		inviteID,
	).Scan(
		&dbSenderID,
		&dbReceiverID,
		&status,
	)
	if err != nil {
		t.Fatal(err)
	}

	if dbSenderID != senderID {
		t.Errorf(
			"expected sender %q, got %q",
			senderID,
			dbSenderID,
		)
	}

	if dbReceiverID != receiverID {
		t.Errorf(
			"expected receiver %q, got %q",
			receiverID,
			dbReceiverID,
		)
	}

	if status != "pending" {
		t.Errorf(
			"expected status %q, got %q",
			"pending",
			status,
		)
	}
}

func TestCreateInvite_InviteConflict(t *testing.T) {
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

	senderID := "11111111-2222-3333-4444-111111111111"
	receiverID := "22222222-3333-4444-5555-222222222222"

	cleanup := func() {
		_, err := db.Exec(
			ctx,
			`
			DELETE FROM match_invites
			WHERE sender_id IN ($1, $2)
			   OR receiver_id IN ($1, $2)
			`,
			senderID,
			receiverID,
		)
		if err != nil {
			t.Errorf("failed to clean invites: %v", err)
		}

		_, err = db.Exec(
			ctx,
			`DELETE FROM users WHERE id IN ($1, $2)`,
			senderID,
			receiverID,
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
			($1, 'Sender'),
			($2, 'Receiver')
		`,
		senderID,
		receiverID,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = repo.CreateInvite(
		ctx,
		senderID,
		receiverID,
	)
	if err != nil {
		t.Fatalf(
			"failed to create first invite: %v",
			err,
		)
	}

	_, err = repo.CreateInvite(
		ctx,
		receiverID,
		senderID,
	)

	if !errors.Is(err, ErrInviteConflict) {
		t.Fatalf(
			"expected ErrInviteConflict, got %v",
			err,
		)
	}
}

func TestIsInvited(t *testing.T) {
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

	senderID := "11111111-2222-3333-4444-111111111111"
	receiverID := "22222222-3333-4444-5555-222222222222"
	inviteID := "33333333-4444-5555-6666-333333333333"

	cleanup := func() {
		_, err := db.Exec(
			ctx,
			`DELETE FROM match_invites WHERE id = $1`,
			inviteID,
		)
		if err != nil {
			t.Errorf("failed to clean invite: %v", err)
		}

		_, err = db.Exec(
			ctx,
			`DELETE FROM users WHERE id IN ($1, $2)`,
			senderID,
			receiverID,
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
			($1, 'Sender'),
			($2, 'Receiver')
		`,
		senderID,
		receiverID,
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
		VALUES ($1, $2, $3, 'pending')
		`,
		inviteID,
		senderID,
		receiverID,
	)
	if err != nil {
		t.Fatal(err)
	}

	exists, err := repo.IsInvited(
		ctx,
		senderID,
		receiverID,
	)
	if err != nil {
		t.Fatal(err)
	}

	if !exists {
		t.Error("expected pending invite to exist")
	}

	exists, err = repo.IsInvited(
		ctx,
		receiverID,
		senderID,
	)
	if err != nil {
		t.Fatal(err)
	}

	if !exists {
		t.Error(
			"expected pending invite to exist regardless of direction",
		)
	}
}

func TestGetPendingInvites(t *testing.T) {
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

	senderID := "11111111-2222-3333-4444-111111111111"
	receiverID := "22222222-3333-4444-5555-222222222222"
	otherUserID := "33333333-4444-5555-6666-333333333333"

	pendingInviteID := "aaaaaaaa-1111-2222-3333-aaaaaaaaaaaa"
	sentInviteID := "bbbbbbbb-2222-3333-4444-bbbbbbbbbbbb"
	rejectedInviteID := "cccccccc-3333-4444-5555-cccccccccccc"

	cleanup := func() {
		_, err := db.Exec(
			ctx,
			`
			DELETE FROM match_invites
			WHERE sender_id IN ($1, $2, $3)
			   OR receiver_id IN ($1, $2, $3)
			`,
			senderID,
			receiverID,
			otherUserID,
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
			senderID,
			receiverID,
			otherUserID,
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
			($1, 'Sender'),
			($2, 'Receiver'),
			($3, 'Other User')
		`,
		senderID,
		receiverID,
		otherUserID,
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
			($4, $3, $5, 'pending'),
			($6, $2, $5, 'rejected')
		`,
		pendingInviteID,
		senderID,
		receiverID,
		sentInviteID,
		otherUserID,
		rejectedInviteID,
	)
	if err != nil {
		t.Fatal(err)
	}

	invites, err := repo.GetPendingInvites(
		ctx,
		receiverID,
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(invites) != 1 {
		t.Fatalf(
			"expected 1 pending invite, got %d",
			len(invites),
		)
	}

	invite := invites[0]

	if invite.InviteID != pendingInviteID {
		t.Errorf(
			"expected invite ID %q, got %q",
			pendingInviteID,
			invite.InviteID,
		)
	}

	if invite.SenderID != senderID {
		t.Errorf(
			"expected sender ID %q, got %q",
			senderID,
			invite.SenderID,
		)
	}

	if invite.ReceiverID != receiverID {
		t.Errorf(
			"expected receiver ID %q, got %q",
			receiverID,
			invite.ReceiverID,
		)
	}

	if invite.Status != "pending" {
		t.Errorf(
			"expected status %q, got %q",
			"pending",
			invite.Status,
		)
	}
}
