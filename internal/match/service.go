package match

import (
	"context"
	"errors"
)

var (
	ErrMissingUsers   = errors.New("sender and receiver are required")
	ErrSameUser       = errors.New("sender and receiver cannot be the same user")
	ErrAlreadyInvited = errors.New("sender and receiver are already invited")
)

type Service struct {
	Repository *Repository
}

func (s *Service) CreateInvite(ctx context.Context, sender, receiver string) error {
	if sender == "" || receiver == "" {
		return ErrMissingUsers
	}

	if sender == receiver {
		return ErrSameUser
	}

	exists, err := s.IsInvited(ctx, sender, receiver)
	if err != nil {
		return err
	}
	if exists {
		return ErrAlreadyInvited
	}

	err = s.Repository.CreateInvite(ctx, sender, receiver)
	
	if errors.Is(err, ErrInviteConflict) {
		return ErrAlreadyInvited
	}

	return err
}

func (s *Service) IsInvited(ctx context.Context, sender, receiver string) (bool, error) {
	exists, err := s.Repository.IsInvited(ctx, sender, receiver)
	if err != nil {
		return false, err
	}
	return exists, nil
}
