package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"matchmap/internal/location"
)

func TestUnregisterDoesNotRemoveReplacementClient(t *testing.T) {
	hub := NewHub(context.Background())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("failed to accept websocket connection: %v", err)
			return
		}
		defer conn.CloseNow()

		<-r.Context().Done()
	}))

	defer server.Close()

	connect := func(userID string) (*Client, *websocket.Conn) {
		t.Helper()

		url := "ws" + server.URL[len("http"):] + "?user=" + userID

		conn, _, err := websocket.Dial(context.Background(), url, nil)
		if err != nil {
			t.Fatalf("failed to dial websocket: %v", err)
		}

		client := &Client{
			UserID: userID,
			Conn:   conn,
		}

		return client, conn
	}

	oldClient, oldConn := connect("user-1")
	defer oldConn.CloseNow()

	hub.Register(oldClient)

	newClient, newConn := connect("user-1")
	defer newConn.CloseNow()

	hub.Register(newClient)

	if hub.ClientCount() != 1 {
		t.Fatalf("expected 1 registered client, got %d", hub.ClientCount())
	}

	if hub.IsCurrentClient(oldClient) {
		t.Fatal("old client should no longer be registered")
	}

	if !hub.IsCurrentClient(newClient) {
		t.Fatal("new client should be registered")
	}

	hub.Unregister(oldClient)

	if hub.ClientCount() != 1 {
		t.Fatalf("expected 1 registered client after unregistering old client, got %d", hub.ClientCount())
	}

	if !hub.IsCurrentClient(newClient) {
		t.Fatal("unregistering the old client removed the replacement client")
	}
}

func TestRegisterClientWithoutLocation(t *testing.T) {
	hub := NewHub(context.Background())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("failed to accept websocket connection: %v", err)
			return
		}
		defer conn.CloseNow()

		<-r.Context().Done()
	}))

	defer server.Close()

	conn, _, err := websocket.Dial(
		context.Background(),
		"ws"+server.URL[len("http"):],
		nil,
	)
	if err != nil {
		t.Fatalf("failed to dial websocket: %v", err)
	}
	defer conn.CloseNow()

	client := &Client{
		UserID: "user-1",
		Conn:   conn,
	}

	hub.Register(client)

	if !hub.IsCurrentClient(client) {
		t.Fatal("client should be registered")
	}

	if client.LocationUpdated {
		t.Fatal("client should not have a location update")
	}

	nearbyUsers := hub.FindNearbyUsers(client, 1000)
	if len(nearbyUsers) != 0 {
		t.Fatalf("expected no nearby users, got %v", nearbyUsers)
	}
}

func TestClientWithoutLocationIsNotFoundByNearbyUsers(t *testing.T) {
	hub := NewHub(context.Background())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("failed to accept websocket connection: %v", err)
			return
		}
		defer conn.CloseNow()
		<-r.Context().Done()
	}))

	defer server.Close()

	connect := func(userID string) *Client {
		t.Helper()

		url := "ws" + server.URL[len("http"):]
		conn, _, err := websocket.Dial(context.Background(), url, nil)
		if err != nil {
			t.Fatalf("failed to dial websocket: %v", err)
		}

		client := &Client{
			UserID: userID,
			Conn:   conn,
		}

		t.Cleanup(func() {
			conn.CloseNow()
		})

		return client
	}

	locatedClient := connect("user-1")
	locatedClient.Location = location.Coordinates{
		Latitude:  -10.9472,
		Longitude: -37.0731,
	}
	locatedClient.LocationUpdated = true

	unlocatedClient := connect("user-2")

	hub.Register(locatedClient)
	hub.Register(unlocatedClient)

	nearbyUsers := hub.FindNearbyUsers(locatedClient, 1000)

	for _, userID := range nearbyUsers {
		if userID == unlocatedClient.UserID {
			t.Fatal("client without location should not appear among nearby users")
		}
	}
}

func TestUpdateClientLocation(t *testing.T) {
	hub := NewHub(context.Background())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("failed to accept websocket connection: %v", err)
			return
		}
		defer conn.CloseNow()
		<-r.Context().Done()
	}))

	defer server.Close()

	conn, _, err := websocket.Dial(
		context.Background(),
		"ws"+server.URL[len("http"):],
		nil,
	)
	if err != nil {
		t.Fatalf("failed to dial websocket: %v", err)
	}
	defer conn.CloseNow()

	client := &Client{
		UserID: "user-1",
		Conn:   conn,
	}

	hub.Register(client)

	if client.LocationUpdated {
		t.Fatal("client should initially have no location")
	}

	coordinates := location.Coordinates{
		Latitude:  -10.9472,
		Longitude: -37.0731,
	}

	updated := hub.UpdateClientLocation(client, coordinates)
	if !updated {
		t.Fatal("expected location update to succeed")
	}

	if !client.LocationUpdated {
		t.Fatal("client should be marked as having an updated location")
	}

	if client.Location != coordinates {
		t.Fatalf("expected location %v, got %v", coordinates, client.Location)
	}
}

func TestUpdateClientLocationRejectsOldConnection(t *testing.T) {
	hub := NewHub(context.Background())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("failed to accept websocket connection: %v", err)
			return
		}
		defer conn.CloseNow()
		<-r.Context().Done()
	}))

	defer server.Close()

	connect := func(userID string) *Client {
		t.Helper()

		url := "ws" + server.URL[len("http"):]
		conn, _, err := websocket.Dial(context.Background(), url, nil)
		if err != nil {
			t.Fatalf("failed to dial websocket: %v", err)
		}

		client := &Client{
			UserID: userID,
			Conn:   conn,
		}

		t.Cleanup(func() {
			conn.CloseNow()
		})

		return client
	}

	oldClient := connect("user-1")
	hub.Register(oldClient)

	newClient := connect("user-1")
	hub.Register(newClient)

	coordinates := location.Coordinates{
		Latitude:  -10.9472,
		Longitude: -37.0731,
	}

	updated := hub.UpdateClientLocation(oldClient, coordinates)
	if updated {
		t.Fatal("expected location update from old connection to be rejected")
	}

	if oldClient.LocationUpdated {
		t.Fatal("old client should not have its location updated")
	}

	if !hub.IsCurrentClient(newClient) {
		t.Fatal("new client should remain registered")
	}
}

func TestUpdateNearbyUsersAddsNearbyClients(t *testing.T) {
	hub := NewHub(context.Background())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("failed to accept websocket connection: %v", err)
			return
		}
		defer conn.CloseNow()
		<-r.Context().Done()
	}))

	defer server.Close()

	connect := func(userID string) *Client {
		t.Helper()

		url := "ws" + server.URL[len("http"):]
		conn, _, err := websocket.Dial(context.Background(), url, nil)
		if err != nil {
			t.Fatalf("failed to dial websocket: %v", err)
		}

		client := &Client{
			UserID: userID,
			Conn:   conn,
		}

		t.Cleanup(func() {
			conn.CloseNow()
		})

		return client
	}

	client1 := connect("user-1")
	client2 := connect("user-2")

	hub.Register(client1)
	hub.Register(client2)

	location1 := location.Coordinates{
		Latitude:  -10.9472,
		Longitude: -37.0731,
	}

	if !hub.UpdateClientLocation(client1, location1) {
		t.Fatal("expected first client's location update to succeed")
	}

	hub.UpdateNearbyUsers(context.Background(), client1, 1000)

	if nearby := hub.FindNearbyUsers(client1, 1000); len(nearby) != 0 {
		t.Fatalf("expected no nearby users, got %v", nearby)
	}

	location2 := location.Coordinates{
		Latitude:  -10.9480,
		Longitude: -37.0731,
	}

	if !hub.UpdateClientLocation(client2, location2) {
		t.Fatal("expected second client's location update to succeed")
	}

	nearby1 := hub.FindNearbyUsers(client1, 1000)
	if !containsUser(nearby1, client2.UserID) {
		t.Fatalf("expected %s among nearby users of %s, got %v",
			client2.UserID, client1.UserID, nearby1)
	}

	nearby2 := hub.FindNearbyUsers(client2, 1000)
	if !containsUser(nearby2, client1.UserID) {
		t.Fatalf("expected %s among nearby users of %s, got %v",
			client1.UserID, client2.UserID, nearby2)
	}
}

func containsUser(users []string, userID string) bool {
	for _, id := range users {
		if id == userID {
			return true
		}
	}
	return false
}

func TestUpdateNearbyUsersRemovesDistantClients(t *testing.T) {
	hub := NewHub(context.Background())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("failed to accept websocket connection: %v", err)
			return
		}
		defer conn.CloseNow()
		<-r.Context().Done()
	}))

	defer server.Close()

	connect := func(userID string) *Client {
		t.Helper()

		url := "ws" + server.URL[len("http"):]
		conn, _, err := websocket.Dial(context.Background(), url, nil)
		if err != nil {
			t.Fatalf("failed to dial websocket: %v", err)
		}

		client := &Client{
			UserID: userID,
			Conn:   conn,
		}

		t.Cleanup(func() {
			conn.CloseNow()
		})

		return client
	}

	client1 := connect("user-1")
	client2 := connect("user-2")

	hub.Register(client1)
	hub.Register(client2)

	location1 := location.Coordinates{
		Latitude:  -10.9472,
		Longitude: -37.0731,
	}
	location2 := location.Coordinates{
		Latitude:  -10.9480,
		Longitude: -37.0731,
	}

	if !hub.UpdateClientLocation(client1, location1) {
		t.Fatal("expected first client's location update to succeed")
	}
	if !hub.UpdateClientLocation(client2, location2) {
		t.Fatal("expected second client's location update to succeed")
	}

	hub.UpdateNearbyUsers(context.Background(), client1, 1000)
	hub.UpdateNearbyUsers(context.Background(), client2, 1000)

	if !containsUser(hub.FindNearbyUsers(client1, 1000), client2.UserID) {
		t.Fatal("expected clients to be nearby before moving")
	}
	if !containsUser(hub.FindNearbyUsers(client2, 1000), client1.UserID) {
		t.Fatal("expected nearby relationship to be reciprocal before moving")
	}

	distantLocation := location.Coordinates{
		Latitude:  -10.9700,
		Longitude: -37.0731,
	}

	if !hub.UpdateClientLocation(client2, distantLocation) {
		t.Fatal("expected second client's location update to succeed")
	}

	hub.UpdateNearbyUsers(context.Background(), client2, 1000)

	if containsUser(hub.FindNearbyUsers(client1, 1000), client2.UserID) {
		t.Fatal("distant client should no longer appear in first client's nearby list")
	}
	if containsUser(hub.FindNearbyUsers(client2, 1000), client1.UserID) {
		t.Fatal("first client should no longer appear in distant client's nearby list")
	}
}

func TestUpdateNearbyUsersNotifiesWhenClientLeaves(t *testing.T) {
	hub := NewHub(context.Background())

	acceptedConns := make(chan *websocket.Conn, 2)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverConn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("failed to accept websocket connection: %v", err)
			return
		}
		defer serverConn.CloseNow()

		acceptedConns <- serverConn

		<-r.Context().Done()
	}))
	defer server.Close()

	connect := func(userID string) (*Client, *websocket.Conn) {
		t.Helper()

		url := "ws" + server.URL[len("http"):]

		clientConn, _, err := websocket.Dial(context.Background(), url, nil)
		if err != nil {
			t.Fatalf("failed to dial websocket: %v", err)
		}

		var serverConn *websocket.Conn

		select {
		case serverConn = <-acceptedConns:
		case <-time.After(2 * time.Second):
			clientConn.CloseNow()
			t.Fatal("timed out waiting for server-side websocket connection")
		}

		client := &Client{
			UserID: userID,
			Conn:   serverConn,
		}

		t.Cleanup(func() {
			clientConn.CloseNow()
			serverConn.CloseNow()
		})

		return client, clientConn
	}

	client1, conn1 := connect("user-1")
	client2, _ := connect("user-2")

	hub.Register(client1)
	hub.Register(client2)

	location1 := location.Coordinates{
		Latitude:  -10.9472,
		Longitude: -37.0731,
	}
	location2 := location.Coordinates{
		Latitude:  -10.9480,
		Longitude: -37.0731,
	}

	if !hub.UpdateClientLocation(client1, location1) {
		t.Fatal("expected first client's location update to succeed")
	}

	if !hub.UpdateClientLocation(client2, location2) {
		t.Fatal("expected second client's location update to succeed")
	}

	hub.UpdateNearbyUsers(context.Background(), client1, 1000)
	hub.UpdateNearbyUsers(context.Background(), client2, 1000)

	previousNearby := hub.FindNearbyUsers(client2, 1000)
	if !containsUser(previousNearby, client1.UserID) {
		t.Fatalf(
			"expected %s in %s's nearby list before moving, got %v",
			client1.UserID,
			client2.UserID,
			previousNearby,
		)
	}

	distantLocation := location.Coordinates{
		Latitude:  -10.9700,
		Longitude: -37.0731,
	}

	if !hub.UpdateClientLocation(client2, distantLocation) {
		t.Fatal("expected second client's location update to succeed")
	}

	hub.UpdateNearbyUsers(context.Background(), client2, 1000)

	updatedNearby := hub.FindNearbyUsers(client2, 1000)
	if containsUser(updatedNearby, client1.UserID) {
		t.Fatalf(
			"expected %s to be removed from %s's nearby list, got %v",
			client1.UserID,
			client2.UserID,
			updatedNearby,
		)
	}

	readCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var event Event

	for {
		if err := wsjson.Read(readCtx, conn1, &event); err != nil {
			t.Fatalf("failed to read user_left notification: %v", err)
		}

		if event.Type == "user_left" {
			break
		}
	}

	var payload UserLeftNotification
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("failed to decode user_left payload: %v", err)
	}

	if payload.UserID != client2.UserID {
		t.Fatalf(
			"expected user_id %q, got %q",
			client2.UserID,
			payload.UserID,
		)
	}
}

func TestBroadcastSendsMessageToAllClients(t *testing.T) {
	hub := NewHub(context.Background())

	acceptedConns := make(chan *websocket.Conn, 2)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("failed to accept websocket connection: %v", err)
			return
		}

		acceptedConns <- conn

		<-r.Context().Done()
		_ = conn.CloseNow()
	}))
	defer server.Close()

	connect := func(userID string) (*Client, *websocket.Conn) {
		t.Helper()

		url := "ws" + server.URL[len("http"):] + "?user=" + userID

		clientConn, _, err := websocket.Dial(context.Background(), url, nil)
		if err != nil {
			t.Fatalf("failed to dial websocket: %v", err)
		}

		t.Cleanup(func() {
			_ = clientConn.CloseNow()
		})

		var serverConn *websocket.Conn

		select {
		case serverConn = <-acceptedConns:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for server connection")
		}

		client := &Client{
			UserID: userID,
			Conn:   serverConn,
		}

		return client, clientConn
	}

	client1, conn1 := connect("user-1")
	client2, conn2 := connect("user-2")

	hub.Register(client1)
	hub.Register(client2)

	sentAt := time.Now().UTC().Truncate(time.Second)

	message := Message{
		User:    "user-1",
		Content: "Hello, everyone!",
		SentAt:  sentAt,
	}

	hub.Broadcast(context.Background(), message)

	connections := map[string]*websocket.Conn{
		"user-1": conn1,
		"user-2": conn2,
	}

	for userID, conn := range connections {
		t.Run(userID, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(
				context.Background(),
				2*time.Second,
			)
			defer cancel()

			var received Message

			err := wsjson.Read(ctx, conn, &received)
			if err != nil {
				t.Fatalf("failed to receive broadcast message: %v", err)
			}

			if received.User != message.User {
				t.Errorf(
					"expected User %q, got %q",
					message.User,
					received.User,
				)
			}

			if received.Content != message.Content {
				t.Errorf(
					"expected Content %q, got %q",
					message.Content,
					received.Content,
				)
			}

			if !received.SentAt.Equal(message.SentAt) {
				t.Errorf(
					"expected SentAt %v, got %v",
					message.SentAt,
					received.SentAt,
				)
			}
		})
	}
}

func TestUnregisterRemovesRegisteredClient(t *testing.T) {
	hub := NewHub(context.Background())

	client := &Client{
		UserID: "user-1",
	}

	hub.Register(client)
	hub.Unregister(client)

	if hub.ClientCount() != 0 {
		t.Fatalf("expected 0 registered clients, got %d", hub.ClientCount())
	}

	if _, exists := hub.Clients[client.UserID]; exists {
		t.Fatal("client should have been removed from Clients")
	}

	if _, exists := hub.Nearby[client.UserID]; exists {
		t.Fatal("client should have been removed from Nearby")
	}
}

func TestUnregisterNotifiesNearbyUsers(t *testing.T) {
	hub := NewHub(context.Background())

	acceptedConns := make(chan *websocket.Conn, 2)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("failed to accept websocket connection: %v", err)
			return
		}

		acceptedConns <- conn

		<-r.Context().Done()
		_ = conn.CloseNow()
	}))
	defer server.Close()

	connect := func(userID string) (*Client, *websocket.Conn) {
		t.Helper()

		url := "ws" + server.URL[len("http"):] + "?user=" + userID

		clientConn, _, err := websocket.Dial(context.Background(), url, nil)
		if err != nil {
			t.Fatalf("failed to dial websocket: %v", err)
		}

		t.Cleanup(func() {
			_ = clientConn.CloseNow()
		})

		var serverConn *websocket.Conn

		select {
		case serverConn = <-acceptedConns:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for server connection")
		}

		return &Client{
			UserID: userID,
			Conn:   serverConn,
		}, clientConn
	}

	client1, _ := connect("user-1")
	client2, client2Conn := connect("user-2")

	client1.Location = location.Coordinates{
		Latitude:  0,
		Longitude: 0,
	}
	client1.LocationUpdated = true

	client2.Location = location.Coordinates{
		Latitude:  0,
		Longitude: 0.001,
	}
	client2.LocationUpdated = true

	hub.Register(client1)
	hub.Register(client2)

	hub.UpdateNearbyUsers(context.Background(), client1, 1000)

	hub.Unregister(client1)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	for {
		var event Event

		err := wsjson.Read(ctx, client2Conn, &event)
		if err != nil {
			t.Fatalf("failed to receive user_left event: %v", err)
		}

		if event.Type != "user_left" {
			continue
		}

		var notification UserLeftNotification

		if err := json.Unmarshal(event.Payload, &notification); err != nil {
			t.Fatalf("failed to decode user_left payload: %v", err)
		}

		if notification.UserID != client1.UserID {
			t.Errorf(
				"expected user_id %q, got %q",
				client1.UserID,
				notification.UserID,
			)
		}

		break
	}

	if hub.IsCurrentClient(client1) {
		t.Error("unregistered client should no longer be current")
	}
}

func TestFindNearbyUsersRespectsRadius(t *testing.T) {
	hub := NewHub(context.Background())

	origin := &Client{
		UserID: "user-1",
		Location: location.Coordinates{
			Latitude:  -10.9472,
			Longitude: -37.0731,
		},
		LocationUpdated: true,
	}

	nearbyClient := &Client{
		UserID: "user-2",
		Location: location.Coordinates{
			Latitude:  -10.9480,
			Longitude: -37.0731,
		},
		LocationUpdated: true,
	}

	distantClient := &Client{
		UserID: "user-3",
		Location: location.Coordinates{
			Latitude:  -10.9600,
			Longitude: -37.0731,
		},
		LocationUpdated: true,
	}

	hub.Register(origin)
	hub.Register(nearbyClient)
	hub.Register(distantClient)

	nearbyUsers := hub.FindNearbyUsers(origin, 1000)

	foundNearby := false
	foundDistant := false
	foundSelf := false

	for _, userID := range nearbyUsers {
		switch userID {
		case nearbyClient.UserID:
			foundNearby = true
		case distantClient.UserID:
			foundDistant = true
		case origin.UserID:
			foundSelf = true
		}
	}

	if !foundNearby {
		t.Error("expected nearby client to be included")
	}

	if foundDistant {
		t.Error("expected distant client to be excluded")
	}

	if foundSelf {
		t.Error("expected the requesting client to be excluded")
	}
}

func TestCompareNearbyUsers(t *testing.T) {
	hub := NewHub(context.Background())

	client := &Client{
		UserID: "user-1",
	}

	hub.Nearby[client.UserID] = []string{
		"user-2",
		"user-3",
	}

	updatedNearby := []string{
		"user-3",
		"user-4",
	}

	newNearby, noLongerNearby, stillNearby := hub.CompareNearbyUsers(
		client,
		updatedNearby,
	)

	assertUserIDs := func(name string, got, want []string) {
		t.Helper()

		if len(got) != len(want) {
			t.Fatalf("%s: expected %v, got %v", name, want, got)
		}

		gotSet := make(map[string]bool)
		for _, userID := range got {
			gotSet[userID] = true
		}

		for _, userID := range want {
			if !gotSet[userID] {
				t.Errorf("%s: expected user %q in %v", name, userID, got)
			}
		}
	}

	assertUserIDs("newNearby", newNearby, []string{"user-4"})
	assertUserIDs("noLongerNearby", noLongerNearby, []string{"user-2"})
	assertUserIDs("stillNearby", stillNearby, []string{"user-3"})
}

func TestUpdateClientLocationRejectsUnregisteredClient(t *testing.T) {
	hub := NewHub(context.Background())

	client := &Client{
		UserID: "user-1",
	}

	coordinates := location.Coordinates{
		Latitude:  -10.9472,
		Longitude: -37.0731,
	}

	updated := hub.UpdateClientLocation(client, coordinates)

	if updated {
		t.Fatal("expected location update for unregistered client to be rejected")
	}

	if client.LocationUpdated {
		t.Fatal("unregistered client's location should not be marked as updated")
	}

	if client.Location != (location.Coordinates{}) {
		t.Fatalf("expected location to remain unchanged, got %v", client.Location)
	}

	if hub.ClientCount() != 0 {
		t.Fatalf("expected 0 registered clients, got %d", hub.ClientCount())
	}
}
