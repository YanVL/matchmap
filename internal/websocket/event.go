package websocket

import "encoding/json"

type Event struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type MatchInviteNotification struct {
	InviteID string `json:"invite_id"`
	SenderID string `json:"sender_id"`
}

type MatchAcceptedNotification struct {
	MatchID    string `json:"match_id"`
	PlayerID   string `json:"player_id"`
	OpponentID string `json:"opponent_id"`
}

type MatchRejectedNotification struct {
	UserID  string `json:"user_id"`
	InviteID string `json:"invite_id"`
}

type MatchFinishedNotification struct {
	MatchID string `json:"match_id"`
}

type MatchResultNotification struct {
	MatchID     string `json:"match_id"`
	PlayerID    string `json:"player_id"`
	MatchResult string `json:"match_result"`
}

type LocationUpdate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type NearbyUserNotification struct {
	UserID    string  `json:"user_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type UserLeftNotification struct {
	UserID string `json:"user_id"`
}

type UserLocationUpdateNotification struct {
	UserID    string  `json:"user_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
