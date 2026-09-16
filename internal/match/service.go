package match

import (
	"context"
	"errors"
)

var (
	ErrMissingUsers       = errors.New("sender and receiver are required")
	ErrSameUser           = errors.New("sender and receiver cannot be the same user")
	ErrAlreadyInvited     = errors.New("sender and receiver are already invited")
	ErrUserAlreadyInMatch = errors.New("one of the users is already in a match")
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

	err = s.ensureUsersNotInMatch(ctx, sender, receiver)
	if err != nil {
		return "", err
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

	invite, err := s.Repository.GetInviteByID(ctx, inviteID)
	if err != nil {
		return err
	}

	if err := s.ensureUsersNotInMatch(ctx, invite.SenderID, invite.ReceiverID); err != nil {
		return err
	}

	return s.Repository.AcceptInviteAndCreateMatch(ctx, inviteID)
}

func (s *Service) ensureUsersNotInMatch(ctx context.Context, sender, receiver string) error {
	senderInMatch, err := s.Repository.IsUserInMatch(ctx, sender)
	if err != nil {
		return err
	}
	if senderInMatch {
		return ErrUserAlreadyInMatch
	}

	receiverInMatch, err := s.Repository.IsUserInMatch(ctx, receiver)
	if err != nil {
		return err
	}
	if receiverInMatch {
		return ErrUserAlreadyInMatch
	}

	return nil
}

func (s *Service) RejectInvite(ctx context.Context, inviteID string) error {
	return s.Repository.RejectInvite(ctx, inviteID)
}
