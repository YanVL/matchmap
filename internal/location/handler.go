package location

import (
	"encoding/json"
	"net/http"
)

type Handler struct {
	Service *Service
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

func (h *Handler) Nearby(w http.ResponseWriter, r *http.Request) {
	var input NearbyRequest

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	users, err := h.Service.Nearby(r.Context(), input.Longitude, input.Latitude, input.Radius)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (h *Handler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	var input UpdateLocationRequest

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	err := h.Service.UpdateLocation(r.Context(), r.PathValue("id"), input.Latitude, input.Longitude)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
