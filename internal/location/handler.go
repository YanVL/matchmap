package location

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	DB *pgxpool.Pool
}

type NearbyRequest struct {
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
	Radius    int     `json:"radius"`
}

type UpdateLocationRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func (h Handler) Nearby(w http.ResponseWriter, r *http.Request) {
	var input NearbyRequest

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	rows, err := h.DB.Query(
		r.Context(),
		`
		SELECT user_id
		FROM user_locations
		WHERE ST_DWithin(
			location,
			ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
			$3
		)
		`,
		input.Longitude,
		input.Latitude,
		input.Radius,
	)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []string

	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			http.Error(w, "scan error", http.StatusInternalServerError)
			return
		}

		users = append(users, userID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (h Handler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	var input UpdateLocationRequest

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	userID := r.PathValue("id")

	_, err := h.DB.Exec(
		r.Context(),
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
		input.Longitude,
		input.Latitude,
		userID,
	)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
