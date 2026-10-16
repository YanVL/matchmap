package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"matchmap/internal/location"
	"sync"

	"github.com/coder/websocket/wsjson"
)

type Hub struct {
	mu      sync.RWMutex
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
	h.mu.Lock()

	oldClient := h.Clients[client.UserID]
	var previousNearby []string

	if oldClient != nil && oldClient != client {
		previousNearby = append([]string(nil), h.Nearby[client.UserID]...)
	}

	h.Clients[client.UserID] = client

	if !client.LocationUpdated {
		h.Nearby[client.UserID] = nil
	} else {
		h.Nearby[client.UserID] = h.findNearbyUsersLocked(client, 1000)
	}

	h.mu.Unlock()

	if oldClient != nil && oldClient != client {
		oldClient.Conn.CloseNow()

		log.Printf(
			"Previous connection closed: user_id=%s connection=%p",
			client.UserID,
			oldClient.Conn,
		)
	}

	if oldClient != nil && oldClient != client {
		h.notifyUsersLeft(h.Ctx, client, previousNearby)
	}

	log.Printf(
		"User %s connected. Nearby users: %v",
		client.UserID,
		h.Nearby[client.UserID],
	)
}

func (h *Hub) Unregister(client *Client) {

	h.mu.Lock()

	currentClient, exists := h.Clients[client.UserID]
	if !exists || currentClient != client {
		h.mu.Unlock()
		return
	}

	nearbyUsersIDs := append([]string(nil), h.Nearby[client.UserID]...)

	delete(h.Clients, client.UserID)
	delete(h.Nearby, client.UserID)

	h.mu.Unlock()

	for _, nearbyUserID := range nearbyUsersIDs {
		h.mu.RLock()
		otherClient, exists := h.Clients[nearbyUserID]
		h.mu.RUnlock()

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
}

func (h *Hub) Broadcast(ctx context.Context, message Message) {

	h.mu.RLock()

	clients := make([]*Client, 0, len(h.Clients))

	for _, client := range h.Clients {
		clients = append(clients, client)
	}

	h.mu.RUnlock()

	for _, client := range clients {

		log.Printf("sending message to client %s: %+v", client.UserID, message)

		err := wsjson.Write(ctx, client.Conn, message)
		if err != nil {
			log.Printf("failed to send message to client %s: %v", client.UserID, err)
		}
	}
}

func (h *Hub) FindNearbyUsers(client *Client, radius float64) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.findNearbyUsersLocked(client, radius)
}

func (h *Hub) findNearbyUsersLocked(client *Client, radius float64) []string {
	var nearbyUserIDs []string

	if !client.LocationUpdated {
		return nearbyUserIDs
	}

	for _, otherClient := range h.Clients {
		if otherClient.UserID == client.UserID {
			continue
		}

		if !otherClient.LocationUpdated {
			continue
		}

		if location.IsNearby(client.Location, otherClient.Location, radius) {
			nearbyUserIDs = append(nearbyUserIDs, otherClient.UserID)
		}
	}

	return nearbyUserIDs
}

func (h *Hub) UpdateClientLocation(client *Client, coordinates location.Coordinates) {
	h.mu.Lock()
	defer h.mu.Unlock()

	client.Location = coordinates
	client.LocationUpdated = true
}

func (h *Hub) UpdateNearbyUsers(ctx context.Context, client *Client, radius float64) {

	h.mu.Lock()

	nearbyUsers := h.findNearbyUsersLocked(client, radius)

	newNearby, noLongerNearby, stillNearby := h.compareNearbyUsersLocked(client, nearbyUsers)

	h.Nearby[client.UserID] = nearbyUsers

	for _, otherClient := range h.Clients {
		if otherClient.UserID == client.UserID {
			continue
		}

		h.Nearby[otherClient.UserID] = h.findNearbyUsersLocked(otherClient, radius)
	}

	h.mu.Unlock()

	h.NotifyNearbyUsers(ctx, client, newNearby)
	h.notifyUsersLeft(ctx, client, noLongerNearby)
	h.NotifyLocationUpdate(ctx, client, stillNearby)

	log.Printf("Updated nearby users for %s: %v", client.UserID, nearbyUsers)
}

func (h *Hub) CompareNearbyUsers(client *Client, nearbyUsers []string) (newNearby []string, noLongerNearby []string, stillNearby []string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.compareNearbyUsersLocked(client, nearbyUsers)
}

func (h *Hub) compareNearbyUsersLocked(client *Client, nearbyUsers []string) (newNearby []string, noLongerNearby []string, stillNearby []string) {
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

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.Clients)
}

// Notification functions
func (h *Hub) NotifyNearbyUsers(ctx context.Context, client *Client, newNearby []string) {

	for _, userID := range newNearby {

		h.mu.RLock()
		otherClient, exists := h.Clients[userID]
		coordinates := client.Location
		h.mu.RUnlock()

		if !exists {
			continue
		}

		notification := NearbyUserNotification{
			UserID:    client.UserID,
			Latitude:  coordinates.Latitude,
			Longitude: coordinates.Longitude,
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

		h.mu.RLock()
		otherClient, exists := h.Clients[userID]
		h.mu.RUnlock()

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

		h.mu.RLock()
		otherClient, exists := h.Clients[userID]
		coordinates := client.Location
		h.mu.RUnlock()

		if !exists {
			continue
		}

		notification := UserLocationUpdateNotification{
			UserID:    client.UserID,
			Latitude:  coordinates.Latitude,
			Longitude: coordinates.Longitude,
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

	h.mu.RLock()
	otherClient, exists := h.Clients[userID]
	h.mu.RUnlock()

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

	h.mu.RLock()
	otherClient, exists := h.Clients[userID]
	h.mu.RUnlock()

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

	h.mu.RLock()
	otherClient, exists := h.Clients[userID]
	h.mu.RUnlock()

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

	h.mu.RLock()
	otherClient, exists := h.Clients[userID]
	h.mu.RUnlock()

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

	h.mu.RLock()
	otherClient, exists := h.Clients[recipientID]
	h.mu.RUnlock()

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

func (h *Hub) NotifyMessageReceived(ctx context.Context, recipientID, conversationID, senderID, message string) error {

	h.mu.RLock()
	otherClient, exists := h.Clients[recipientID]
	h.mu.RUnlock()

	if !exists {
		return nil
	}

	payload, err := json.Marshal(PrivateMessageNotification{
		ConversationID: conversationID,
		SenderID:       senderID,
		Content:        message,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal message notification: %w", err)
	}

	event := Event{
		Type:    "message_received",
		Payload: payload,
	}

	return wsjson.Write(ctx, otherClient.Conn, event)
}
