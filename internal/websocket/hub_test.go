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
