package match

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
)

type Handler struct {
	Service *Service
}

const (
	errInternalServer = "internal server error"
	headerContentType = "Content-Type"
	contentTypeJSON   = "application/json"
)

var (
	ErrInvalidUUID    = errors.New("invalid UUID format")
	ErrRequiredFields = errors.New("required fields are missing")
)

func validateUUIDs(ids ...string) error {
	for _, id := range ids {
		if _, err := uuid.Parse(id); err != nil {
			return ErrInvalidUUID
		}
	}

	return nil
}

func (h *Handler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	sender := r.URL.Query().Get("sender")
	receiver := r.URL.Query().Get("receiver")

	inviteID, err := h.Service.CreateInvite(r.Context(), sender, receiver)
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

	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"invite_id": inviteID,
		"message":   "Invite created",
	})
}

func (h *Handler) GetPendingInvites(w http.ResponseWriter, r *http.Request) {

	if r.URL.Query().Get("user_id") == "" {
		http.Error(w, ErrRequiredFields.Error(), http.StatusBadRequest)
		return
	}

	userID := r.URL.Query().Get("user_id")

	if err := validateUUIDs(userID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	invites, err := h.Service.GetPendingInvites(r.Context(), userID)

	if err != nil {
		http.Error(w, errInternalServer, http.StatusInternalServerError)
		return
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"invites": invites,
	})
}

func (h *Handler) AcceptInvite(w http.ResponseWriter, r *http.Request) {

	if r.URL.Query().Get("invite_id") == "" {
		http.Error(w, ErrRequiredFields.Error(), http.StatusBadRequest)
		return
	}

	inviteID := r.URL.Query().Get("invite_id")

	if err := validateUUIDs(inviteID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.Service.AcceptInvite(r.Context(), inviteID)
	if err != nil {
		switch {
		case errors.Is(err, ErrInviteNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, ErrUserAlreadyInMatch):
			http.Error(w, err.Error(), http.StatusConflict)
		default:
			http.Error(w, errInternalServer, http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Invite accepted"))
}

func (h *Handler) RejectInvite(w http.ResponseWriter, r *http.Request) {

	if r.URL.Query().Get("invite_id") == "" {
		http.Error(w, ErrRequiredFields.Error(), http.StatusBadRequest)
		return
	}

	inviteID := r.URL.Query().Get("invite_id")

	if err := validateUUIDs(inviteID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.Service.RejectInvite(r.Context(), inviteID)
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

func (h *Handler) FinishMatch(w http.ResponseWriter, r *http.Request) {

	if r.URL.Query().Get("match_id") == "" || r.URL.Query().Get("user_id") == "" {
		http.Error(w, ErrRequiredFields.Error(), http.StatusBadRequest)
		return
	}

	matchID := r.URL.Query().Get("match_id")
	userID := r.URL.Query().Get("user_id")

	if err := validateUUIDs(matchID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.Service.FinishMatch(r.Context(), matchID, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrMatchNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, errInternalServer, http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Match finish registered"))
}

func (h *Handler) RecordMatchResult(w http.ResponseWriter, r *http.Request) {

	if r.URL.Query().Get("match_id") == "" || r.URL.Query().Get("user_id") == "" || r.URL.Query().Get("result") == "" {
		http.Error(w, ErrRequiredFields.Error(), http.StatusBadRequest)
		return
	}

	matchID := r.URL.Query().Get("match_id")
	userID := r.URL.Query().Get("user_id")
	result := r.URL.Query().Get("result")

	if err := validateUUIDs(matchID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.Service.RecordMatchResult(r.Context(), matchID, userID, result)
	if err != nil {
		switch {
		case errors.Is(err, ErrMatchNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, ErrMatchResultAlreadyRecorded):
			http.Error(w, err.Error(), http.StatusConflict)
		case errors.Is(err, ErrInvalidResult):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrMatchNotFinished):
			http.Error(w, err.Error(), http.StatusConflict)
		default:
			http.Error(w, errInternalServer, http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Match result recorded"))
}

func (h *Handler) GetUserMatchStats(w http.ResponseWriter, r *http.Request) {

	if r.URL.Query().Get("user_id") == "" {
		http.Error(w, ErrRequiredFields.Error(), http.StatusBadRequest)
		return
	}

	userID := r.URL.Query().Get("user_id")

	if err := validateUUIDs(userID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stats, err := h.Service.GetUserMatchStats(r.Context(), userID)
	if err != nil {
		http.Error(w, errInternalServer, http.StatusInternalServerError)
		return
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(stats)
}
