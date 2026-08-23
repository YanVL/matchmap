package websocket

import (
	"context"
	"encoding/json"
	"log"
	"matchmap/internal/location"

	"github.com/coder/websocket/wsjson"
)

type Hub struct {
	Clients map[string]*Client
	Nearby  map[string][]string
}

func NewHub() *Hub {
	return &Hub{
		Clients: make(map[string]*Client),
		Nearby:  make(map[string][]string),
	}
}

func (h *Hub) Register(client *Client) {
	h.Clients[client.UserID] = client

	nearbyUsers := h.FindNearbyUsers(client, 1000)
	h.Nearby[client.UserID] = nearbyUsers

	log.Printf("User %s connected. Nearby users: %v", client.UserID, nearbyUsers)
}

func (h *Hub) Unregister(client *Client) {
	delete(h.Clients, client.UserID)
	delete(h.Nearby, client.UserID)
}

func (h *Hub) Broadcast(ctx context.Context, message Message) {
	for _, client := range h.Clients {

		log.Printf("sending message to client %s: %+v", client.UserID, message)

		err := wsjson.Write(ctx, client.Conn, message)
		if err != nil {
			log.Printf("failed to send message to client %s: %v", client.UserID, err)
		}
	}
}

func (h *Hub) FindNearbyUsers(client *Client, radius float64) []string {
	var nearbyUserIDs []string

	for _, otherClient := range h.Clients {
		if otherClient.UserID == client.UserID {
			continue
		}

		if location.IsNearby(client.Location, otherClient.Location, radius) {
			nearbyUserIDs = append(nearbyUserIDs, otherClient.UserID)
		}
	}

	return nearbyUserIDs
}

func (h *Hub) UpdateNearbyUsers(ctx context.Context, client *Client, radius float64) {
	nearbyUsers := h.FindNearbyUsers(client, radius)

	newNearby, _ := h.CompareNearbyUsers(client, nearbyUsers)

	h.Nearby[client.UserID] = nearbyUsers

	for _, otherClient := range h.Clients {
		if otherClient.UserID == client.UserID {
			continue
		}

		if location.IsNearby(client.Location, otherClient.Location, radius) {
			h.Nearby[otherClient.UserID] = h.FindNearbyUsers(otherClient, radius)
		}
	}

	h.NotifyNearbyUsers(ctx, client, newNearby)

	log.Printf("Updated nearby users for %s: %v", client.UserID, nearbyUsers)
}

func (h *Hub) CompareNearbyUsers(client *Client, nearbyUsers []string) (newNearby []string, noLongerNearby []string) {
	previousNearby := h.Nearby[client.UserID]
	updatedNearby := nearbyUsers

	previousMap := make(map[string]bool)
	for _, userID := range previousNearby {
		previousMap[userID] = true
	}

	updatedMap := make(map[string]bool)
	for _, userID := range updatedNearby {
		updatedMap[userID] = true
	}

	for _, userID := range updatedNearby {
		if !previousMap[userID] {
			newNearby = append(newNearby, userID)
		}
	}

	for _, userID := range previousNearby {
		if !updatedMap[userID] {
			noLongerNearby = append(noLongerNearby, userID)
		}
	}

	return newNearby, noLongerNearby
}

func (h *Hub) NotifyNearbyUsers(ctx context.Context, client *Client, newNearby []string) {
	for _, userID := range newNearby {
		otherClient, exists := h.Clients[userID]
		if !exists {
			continue
		}

		notification := NearbyUserNotification{
			UserID: client.UserID,
		}

		payload, err := json.Marshal(notification)
		if err != nil {
			log.Printf(
				"failed to marshal nearby user notification: %v",
				err,
			)
			continue
		}

		event := Event{
			Type:    "nearby_user",
			Payload: payload,
		}

		err = wsjson.Write(ctx, otherClient.Conn, event)
		if err != nil {
			log.Printf(
				"failed to notify nearby user %s about %s: %v",
				userID,
				client.UserID,
				err,
			)
		}
	}
}
