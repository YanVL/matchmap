package websocket

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func setupClientWriteTest(t *testing.T) (*Client, *websocket.Conn) {
	t.Helper()

	accepted := make(chan *websocket.Conn, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("failed to accept WebSocket connection: %v", err)
			return
		}

		defer conn.CloseNow()

		accepted <- conn

		<-r.Context().Done()
	}))

	t.Cleanup(server.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := "ws" + server.URL[len("http"):]

	clientConn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("failed to establish WebSocket connection: %v", err)
	}

	t.Cleanup(func() {
		_ = clientConn.CloseNow()
	})

	var serverConn *websocket.Conn

	select {
	case serverConn = <-accepted:
	case <-ctx.Done():
		t.Fatal("timed out waiting for server-side WebSocket connection")
	}

	t.Cleanup(func() {
		_ = serverConn.CloseNow()
	})

	client := &Client{
		UserID: "user-1",
		Conn:   clientConn,
	}

	return client, serverConn
}

func TestClientWriteJSON(t *testing.T) {
	client, serverConn := setupClientWriteTest(t)

	message := map[string]string{
		"type":    "chat_message",
		"content": "Hello, world!",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.WriteJSON(ctx, message); err != nil {
		t.Fatalf("WriteJSON returned an unexpected error: %v", err)
	}

	var received map[string]string

	if err := wsjson.Read(ctx, serverConn, &received); err != nil {
		t.Fatalf("failed to read JSON message: %v", err)
	}

	if received["type"] != message["type"] {
		t.Errorf("expected type %q, got %q", message["type"], received["type"])
	}

	if received["content"] != message["content"] {
		t.Errorf("expected content %q, got %q", message["content"], received["content"])
	}
}

func TestClientWriteJSONReturnsErrorWhenConnectionIsClosed(t *testing.T) {
	client, _ := setupClientWriteTest(t)

	if err := client.Conn.CloseNow(); err != nil {
		t.Fatalf("failed to close WebSocket connection: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := client.WriteJSON(ctx, map[string]string{
		"type": "chat_message",
	})

	if err == nil {
		t.Fatal("expected WriteJSON to return an error for a closed connection")
	}
}

func TestClientWriteJSONConcurrentWrites(t *testing.T) {
	client, serverConn := setupClientWriteTest(t)

	const messageCount = 20

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	writeErrors := make(chan error, messageCount)

	var writers sync.WaitGroup
	writers.Add(messageCount)

	for i := 0; i < messageCount; i++ {
		go func(id int) {
			defer writers.Done()

			message := map[string]int{
				"id": id,
			}

			if err := client.WriteJSON(ctx, message); err != nil {
				writeErrors <- fmt.Errorf("write %d failed: %w", id, err)
			}
		}(i)
	}

	receivedIDs := make(map[int]bool, messageCount)

	for i := 0; i < messageCount; i++ {
		var received map[string]int

		if err := wsjson.Read(ctx, serverConn, &received); err != nil {
			t.Fatalf("failed to read message %d: %v", i+1, err)
		}

		id, exists := received["id"]
		if !exists {
			t.Fatalf("message %d does not contain an id: %v", i+1, received)
		}

		if id < 0 || id >= messageCount {
			t.Errorf("received unexpected message id: %d", id)
			continue
		}

		if receivedIDs[id] {
			t.Errorf("received duplicate message id: %d", id)
		}

		receivedIDs[id] = true
	}

	writers.Wait()
	close(writeErrors)

	for err := range writeErrors {
		t.Error(err)
	}

	if len(receivedIDs) != messageCount {
		t.Errorf("expected %d unique messages, received %d", messageCount, len(receivedIDs))
	}
}