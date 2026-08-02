package location

import (
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
	h.Service.Nearby(w, r)
}

func (h *Handler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	h.Service.UpdateLocation(w, r)
}
