package location

import (
	"context"
)

type Service struct {
	Repository *Repository
}

func (s *Service) UpdateLocation(ctx context.Context, userID string, latitude float64, longitude float64) error {
	return s.Repository.SaveLocation(ctx, userID, longitude, latitude)
}

func (s *Service) Nearby(ctx context.Context, longitude, latitude float64, radius int) ([]string, error) {
	return s.Repository.GetNearbyUsers(ctx, longitude, latitude, radius)
}
