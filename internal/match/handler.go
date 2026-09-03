package match

import (
	"errors"
	"net/http"
)

type Handler struct {
	Service *Service
}

func (h *Handler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	sender := r.URL.Query().Get("sender")
	receiver := r.URL.Query().Get("receiver")

	err := h.Service.CreateInvite(r.Context(), sender, receiver)
	if err != nil {
		switch {
		case errors.Is(err, ErrMissingUsers), errors.Is(err, ErrSameUser):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrAlreadyInvited):
			http.Error(w, err.Error(), http.StatusConflict)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Invite created"))
}