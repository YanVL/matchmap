package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"matchmap/internal/database"
	"matchmap/internal/location"
	"matchmap/internal/match"
	"matchmap/internal/users"
	"matchmap/internal/websocket"
)

func main() {
	ctx := context.Background()

	db, err := database.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	locationRepo := location.Repository{
		DB: db,
	}
	locationService := location.Service{
		Repository: &locationRepo,
	}
	locationHandler := location.Handler{
		Service: &locationService,
	}

	userHandler := users.Handler{
		DB: db,
	}
	hub := websocket.NewHub(ctx)
	wsHandler := websocket.Handler{
		Hub:             hub,
		LocationService: &locationService,
	}

	matchRepo := match.Repository{
		DB: db,
	}
	matchService := match.Service{
		Repository: &matchRepo,
		Notifier:   hub,
	}
	matchHandler := match.Handler{
		Service: &matchService,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
	})

	mux.HandleFunc("POST /users", userHandler.CreateUser)
	mux.HandleFunc("PUT /users/{id}/location", locationHandler.UpdateLocation)
	mux.HandleFunc("POST /nearby", locationHandler.Nearby)
	mux.HandleFunc("GET /ws", wsHandler.Connect)
	mux.HandleFunc("POST /match/invite", matchHandler.CreateInvite)
	mux.HandleFunc("GET /match/invites", matchHandler.GetPendingInvites)
	mux.HandleFunc("POST /match/accept", matchHandler.AcceptInvite)
	mux.HandleFunc("POST /match/reject", matchHandler.RejectInvite)
	mux.HandleFunc("POST /match/finish", matchHandler.FinishMatch)
	mux.HandleFunc("POST /match/result", matchHandler.RecordMatchResult)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("api running on port", port)

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
