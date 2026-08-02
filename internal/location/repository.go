package location

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB *pgxpool.Pool
}

func (r *Repository) SaveLocation(ctx context.Context, userID string, longitude, latitude float64) error {
	_, err := r.DB.Exec(
		ctx,
		`
		INSERT INTO user_locations (user_id, location, updated_at)
		VALUES (
			$3,
			ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
			NOW()
		)
		ON CONFLICT (user_id) DO UPDATE SET
			location = EXCLUDED.location,
			updated_at = NOW()
		`,
		longitude,
		latitude,
		userID,
	)

	return err
}

func (r *Repository) GetNearbyUsers(ctx context.Context, longitude, latitude float64, radius int) ([]string, error) {
	rows, err := r.DB.Query(
		ctx,
		`
		SELECT user_id
		FROM user_locations
		WHERE ST_DWithin(
			location,
			ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
			$3
		)
		`,
		longitude,
		latitude,
		radius,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}

	return userIDs, nil
}
