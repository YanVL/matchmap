package chat

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Handler struct {
	Service *Service
}

func (h *Handler) GetOrCreateConversation(w http.ResponseWriter, r *http.Request) {
	user1ID := r.URL.Query().Get("user1_id")
	user2ID := r.URL.Query().Get("user2_id")

	conversationID, err := h.Service.GetOrCreateConversation(r.Context(), user1ID, user2ID)

	if err != nil {
		switch {
		case errors.Is(err, ErrConversationConflict):
			http.Error(w, err.Error(), http.StatusConflict)
		case errors.Is(err, ErrUserIDRequired):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrSameUser):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrNoActiveMatch):
			http.Error(w, err.Error(), http.StatusConflict)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"conversation_id": conversationID,
	})
}

func (h *Handler) CreateMessage(w http.ResponseWriter, r *http.Request) {
	conversationID := r.URL.Query().Get("conversation_id")
	senderID := r.URL.Query().Get("sender_id")
	content := r.URL.Query().Get("content")

	messageID, err := h.Service.CreateMessage(r.Context(), conversationID, senderID, content)

	if err != nil {
		switch {
		case errors.Is(err, ErrMessageIDsRequired):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrEmptyMessage):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrMessageTooLong):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, ErrUserNotInConversation):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, ErrNoActiveMatch):
			http.Error(w, err.Error(), http.StatusForbidden)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message_id": messageID,
	})
}
