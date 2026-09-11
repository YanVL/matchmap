package match

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Handler struct {
	Service *Service
}

const errInternalServer = "internal server error"

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
			http.Error(w, errInternalServer, http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Invite created"))
}

func (h *Handler) GetPendingInvites(w http.ResponseWriter, r *http.Request) {
    userID := r.URL.Query().Get("user_id")

    invites, err := h.Service.GetPendingInvites(r.Context(), userID)

    if err != nil {
        http.Error(w, errInternalServer, http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)

    json.NewEncoder(w).Encode(map[string]interface{}{
        "invites": invites,
    })
}

func (h *Handler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	sender := r.URL.Query().Get("sender")
	receiver := r.URL.Query().Get("receiver")

	err := h.Service.AcceptInvite(r.Context(), sender, receiver)
	if err != nil {
		switch {
		case errors.Is(err, ErrInviteNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, errInternalServer, http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Invite accepted"))
}

func (h *Handler) RejectInvite(w http.ResponseWriter, r *http.Request) {
	sender := r.URL.Query().Get("sender")
	receiver := r.URL.Query().Get("receiver")

	err := h.Service.RejectInvite(r.Context(), sender, receiver)
	if err != nil {
		switch {
		case errors.Is(err, ErrInviteNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, errInternalServer, http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Invite rejected"))
}