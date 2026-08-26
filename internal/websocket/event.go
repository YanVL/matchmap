package websocket

import "encoding/json"

type Event struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type LocationUpdate struct {
    Latitude  float64 `json:"latitude"`
    Longitude float64 `json:"longitude"`
}

type NearbyUserNotification struct {
    UserID string `json:"user_id"`
}

type UserLeftNotification struct {
    UserID string `json:"user_id"`
}