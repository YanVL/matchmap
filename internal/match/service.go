package match

import (
	"context"
	"errors"
)

var (
	ErrMissingUsers               = errors.New("sender and receiver are required")
	ErrSameUser                   = errors.New("sender and receiver cannot be the same user")
	ErrAlreadyInvited             = errors.New("sender and receiver are already invited")
	ErrUserAlreadyInMatch         = errors.New("one of the users is already in a match")
	ErrMatchResultAlreadyRecorded = errors.New("match result has already been recorded")
	ErrInvalidResult              = errors.New("the result must be 'win', 'loss', or 'draw'")
	ErrMatchNotFinished           = errors.New("the match is not finished yet")
	ErrOnePlayerResultMissing     = errors.New("one of the players has not recorded their result yet")
)

type Service struct {
	Repository *Repository
	Notifier   Notifier
}

func (s *Service) CreateInvite(ctx context.Context, sender, receiver string) (string, error) {
	if sender == "" || receiver == "" {
		return "", ErrMissingUsers
	}

	if sender == receiver {
		return "", ErrSameUser
	}

	exists, err := s.IsInvited(ctx, sender, receiver)
	if err != nil {
		return "", err
	}
	if exists {
		return "", ErrAlreadyInvited
	}

	inviteID, err := s.Repository.CreateInvite(ctx, sender, receiver)
	if errors.Is(err, ErrInviteConflict) {
		return "", ErrAlreadyInvited
	}
	if err != nil {
		return "", err
	}

	_ = s.Notifier.NotifyInviteCreated(ctx, receiver, sender)

	return inviteID, nil
}

func (s *Service) IsInvited(ctx context.Context, sender, receiver string) (bool, error) {
	exists, err := s.Repository.IsInvited(ctx, sender, receiver)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (s *Service) GetPendingInvites(ctx context.Context, userID string) ([]MatchInvite, error) {
	invites, err := s.Repository.showPendingInvites(ctx, userID)
	if err != nil {
		return nil, err
	}
	return invites, nil
}

func (s *Service) AcceptInvite(ctx context.Context, inviteID string) error {
	return s.Repository.AcceptInviteAndCreateMatch(ctx, inviteID)
}

func (s *Service) RejectInvite(ctx context.Context, inviteID string) error {
	return s.Repository.RejectInvite(ctx, inviteID)
}

func (s *Service) FinishMatch(ctx context.Context, matchID, userID string) error {
	return s.Repository.FinishMatch(ctx, matchID, userID)
}

func (s *Service) RecordMatchResult(ctx context.Context, matchID, userID, matchResult string) error {

	if matchResult != "win" && matchResult != "loss" && matchResult != "draw" {
		return ErrInvalidResult
	}

	match, err := s.Repository.GetMatchByID(ctx, matchID)
	if err != nil {
		return err
	}

	if match.Player1ID != userID && match.Player2ID != userID {
		return ErrMatchNotFound
	}

	if match.Status != "finished" {
		return ErrMatchNotFinished
	}

	return s.Repository.RecordMatchResult(ctx, matchID, userID, matchResult)
}

type MatchStats struct {
	Matches            int
	Wins               int
	Losses             int
	Draws              int
	Winrate            float64
	ConcordantMatches  int
	DiscordantMatches  int
	ConcordanceRate    float64
	ConcordantWins     int
	WinConcordanceRate float64
	Score              int
}

func (s *Service) GetUserMatchStats(ctx context.Context, userID string) (*MatchStats, error) {
	results, err := s.Repository.GetUserMatchResults(ctx, userID)
	if err != nil {
		return nil, err
	}

	return calculateMatchStats(results), nil
}

func calculateMatchStats(results []UserMatchResult) *MatchStats {
	stats := &MatchStats{}

	for _, match := range results {
		stats.Matches++

		switch match.UserResult {
		case "win":
			stats.Wins++
		case "loss":
			stats.Losses++
		case "draw":
			stats.Draws++
		}

		concordance := "discordant"

		if isConcordant(match.UserResult, match.OpponentResult) {
			concordance = "concordant"
			stats.ConcordantMatches++
		} else {
			stats.DiscordantMatches++
		}

		if match.UserResult == "win" && concordance == "concordant" {
			stats.ConcordantWins++
		}

		stats.Score += calculateMatchScore(
			match.UserResult,
			concordance,
		)
	}

	if stats.Matches > 0 {
		stats.Winrate =
			float64(stats.Wins) /
				float64(stats.Matches) * 100

		stats.ConcordanceRate =
			float64(stats.ConcordantMatches) /
				float64(stats.Matches) * 100
	}

	if stats.Wins > 0 {
		stats.WinConcordanceRate =
			float64(stats.ConcordantWins) /
				float64(stats.Wins) * 100
	}

	return stats
}

func isConcordant(result1, result2 string) bool {
	return (result1 == "win" && result2 == "loss") ||
		(result1 == "loss" && result2 == "win") ||
		(result1 == "draw" && result2 == "draw")
}

func calculateMatchScore(result, concordance string) int {
	score := 0

	switch result {
	case "win":
		score += 3
	case "draw":
		score += 1
	}

	if concordance == "concordant" {
		score += 1
	} else {
		score -= 1
	}

	return score
}
