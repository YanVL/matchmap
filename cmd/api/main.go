package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"matchmap/internal/database"
	"matchmap/internal/location"
	"matchmap/internal/users"
)

func main() {
	ctx := context.Background()

	db, err := database.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	locationHandler := location.Handler{
		DB: db,
	}
	userHandler := users.Handler{
		DB: db,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
	})

	mux.HandleFunc("POST /nearby", locationHandler.Nearby)
	mux.HandleFunc("POST /users", userHandler.CreateUser)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("api running on port", port)

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}