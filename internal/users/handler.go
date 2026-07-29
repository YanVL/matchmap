package users

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	DB *pgxpool.Pool
}

type CreateUserRequest struct {
	Username string `json:"username"`
}

type User struct {
	ID string `json:"id"`
	Name string `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func (h Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var input CreateUserRequest

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if input.Username == "" {
		http.Error(w, "username is required", http.StatusBadRequest)
		return
	}

	var user User

	err := h.DB.QueryRow(
		r.Context(),
		`
		INSERT INTO users (name)
		VALUES ($1)
		RETURNING id, name, created_at
		`,
		input.Username,

	).Scan(
		&user.ID,
		&user.Name,
		&user.CreatedAt,
	)

	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)

}
