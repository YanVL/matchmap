package location

import (
	"encoding/json"
	"net/http"
)

type Service struct {
	Repository *Repository
}

func (s *Service) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	var input UpdateLocationRequest

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	userID := r.PathValue("id")

	err := s.Repository.SaveLocation(r.Context(), userID, input.Longitude, input.Latitude)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Service) Nearby(w http.ResponseWriter, r *http.Request) {

	var input NearbyRequest

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	users, err := s.Repository.GetNearbyUsers(r.Context(), input.Longitude, input.Latitude, input.Radius)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}
