package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"matchmap/internal/location"

	"github.com/coder/websocket/wsjson"
)

type Hub struct {
	Clients map[string]*Client
	Nearby  map[string][]string
	Ctx     context.Context
}

func NewHub(ctx context.Context) *Hub {
	return &Hub{
		Clients: make(map[string]*Client),
		Nearby:  make(map[string][]string),
		Ctx:     ctx,
	}
}

func (h *Hub) Register(client *Client) {
	h.Clients[client.UserID] = client

	nearbyUsers := h.FindNearbyUsers(client, 1000)
	h.Nearby[client.UserID] = nearbyUsers

	log.Printf("User %s connected. Nearby users: %v", client.UserID, nearbyUsers)
}

func (h *Hub) Unregister(client *Client) {

	for _, nearbyUserID := range h.Nearby[client.UserID] {
		otherClient, exists := h.Clients[nearbyUserID]
		if !exists {
			continue
		}

		notification := UserLeftNotification{
			UserID: client.UserID,
		}

		payload, err := json.Marshal(notification)
		if err != nil {
			log.Printf("failed to marshal user left notification: %v", err)
			continue
		}

		event := Event{
			Type:    "user_left",
			Payload: payload,
		}

		err = wsjson.Write(h.Ctx, otherClient.Conn, event)
		if err != nil {
			log.Printf("failed to notify user %s about %s leaving: %v", nearbyUserID, client.UserID, err)
		}
	}

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

	newNearby, noLongerNearby, stillNearby := h.CompareNearbyUsers(client, nearbyUsers)

	h.Nearby[client.UserID] = nearbyUsers

	for _, otherClient := range h.Clients {
		if otherClient.UserID == client.UserID {
			continue
		}

		h.Nearby[otherClient.UserID] = h.FindNearbyUsers(otherClient, radius)
	}

	h.NotifyNearbyUsers(ctx, client, newNearby)
	h.notifyUsersLeft(ctx, client, noLongerNearby)
	h.NotifyLocationUpdate(ctx, client, stillNearby)

	log.Printf("Updated nearby users for %s: %v", client.UserID, nearbyUsers)
}

func (h *Hub) CompareNearbyUsers(client *Client, nearbyUsers []string) (newNearby []string, noLongerNearby []string, stillNearby []string) {
	previousNearby := h.Nearby[client.UserID]
	updatedNearby := nearbyUsers
	stillNearby = make([]string, 0)

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

	for userID := range previousMap {
		if updatedMap[userID] {
			stillNearby = append(stillNearby, userID)
		}
	}

	return newNearby, noLongerNearby, stillNearby
}

func (h *Hub) NotifyNearbyUsers(ctx context.Context, client *Client, newNearby []string) {
	for _, userID := range newNearby {
		otherClient, exists := h.Clients[userID]
		if !exists {
			continue
		}

		notification := NearbyUserNotification{
			UserID:    client.UserID,
			Latitude:  client.Location.Latitude,
			Longitude: client.Location.Longitude,
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

func (h *Hub) notifyUsersLeft(ctx context.Context, client *Client, noLongerNearbyUsers []string) {
	for _, userID := range noLongerNearbyUsers {
		otherClient, exists := h.Clients[userID]
		if !exists {
			continue
		}

		notification := UserLeftNotification{
			UserID: client.UserID,
		}

		payload, err := json.Marshal(notification)
		if err != nil {
			log.Printf(
				"failed to marshal user left notification: %v",
				err,
			)
			continue
		}

		event := Event{
			Type:    "user_left",
			Payload: payload,
		}

		err = wsjson.Write(ctx, otherClient.Conn, event)
		if err != nil {
			log.Printf(
				"failed to notify user %s about %s leaving: %v",
				userID,
				client.UserID,
				err,
			)
		}
	}
}

func (h *Hub) NotifyLocationUpdate(ctx context.Context, client *Client, stillNearby []string) {
	for _, userID := range stillNearby {
		otherClient, exists := h.Clients[userID]
		if !exists {
			continue
		}

		notification := UserLocationUpdateNotification{
			UserID:    client.UserID,
			Latitude:  client.Location.Latitude,
			Longitude: client.Location.Longitude,
		}

		payload, err := json.Marshal(notification)
		if err != nil {
			log.Printf(
				"failed to marshal user location update notification: %v",
				err,
			)
			continue
		}

		event := Event{
			Type:    "user_location_update",
			Payload: payload,
		}

		err = wsjson.Write(ctx, otherClient.Conn, event)
		if err != nil {
			log.Printf(
				"failed to notify user %s about %s location update: %v",
				userID,
				client.UserID,
				err,
			)
		}
	}
}

func (h *Hub) NotifyInviteCreated(ctx context.Context, userID string, senderID string, inviteID string) error {
	otherClient, exists := h.Clients[userID]

	if !exists {
		return nil
	}

	payload, err := json.Marshal(MatchInviteNotification{
		SenderID: senderID,
		InviteID: inviteID,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal match invite: %v", err)
	}

	event := Event{
		Type:    "match_invite",
		Payload: payload,
	}

	return wsjson.Write(ctx, otherClient.Conn, event)
}

func (h *Hub) NotifyMatchAccepted(ctx context.Context, userID, opponentID, matchID string) error {
	otherClient, exists := h.Clients[userID]

	if !exists {
		return nil
	}

	payload, err := json.Marshal(MatchAcceptedNotification{
		MatchID:    matchID,
		PlayerID:   userID,
		OpponentID: opponentID,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal match accepted notification: %v", err)
	}

	event := Event{
		Type:    "match_accepted",
		Payload: payload,
	}

	return wsjson.Write(ctx, otherClient.Conn, event)
}

func (h *Hub) NotifyMatchRejected(ctx context.Context, userID, inviteID string) error {

	otherClient, exists := h.Clients[userID]

	if !exists {
		return nil
	}

	payload, err := json.Marshal(MatchRejectedNotification{
		UserID:   userID,
		InviteID: inviteID,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal match rejected notification: %v", err)
	}

	event := Event{
		Type:    "match_rejected",
		Payload: payload,
	}

	return wsjson.Write(ctx, otherClient.Conn, event)
}

func (h *Hub) NotifyMatchFinished(ctx context.Context, userID, opponentID, matchID string) error {
	otherClient, exists := h.Clients[userID]

	if !exists {
		return nil
	}

	payload, err := json.Marshal(MatchFinishedNotification{
		MatchID: matchID,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal match finished notification: %v", err)
	}

	event := Event{
		Type:    "match_finished",
		Payload: payload,
	}

	return wsjson.Write(ctx, otherClient.Conn, event)
}

func (h *Hub) NotifyMatchResult(ctx context.Context, recipientID, playerID, matchID, matchResult string) error {
	otherClient, exists := h.Clients[recipientID]

	if !exists {
		return nil
	}

	payload, err := json.Marshal(MatchResultNotification{
		MatchID:     matchID,
		PlayerID:    playerID,
		MatchResult: matchResult,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal match result notification: %v", err)
	}

	event := Event{
		Type:    "match_result",
		Payload: payload,
	}

	return wsjson.Write(ctx, otherClient.Conn, event)
}
