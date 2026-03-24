package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/parus/distributed-url-shortener/internal/cache"
	"github.com/parus/distributed-url-shortener/internal/database"
	"github.com/parus/distributed-url-shortener/internal/handler"
	"github.com/parus/distributed-url-shortener/internal/middleware"
	"github.com/parus/distributed-url-shortener/internal/service"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting Distributed URL Shortener...")

	// Initialize PostgreSQL.
	db, err := database.NewPostgresStore()
	if err != nil {
		log.Fatalf("Failed to initialize PostgreSQL: %v", err)
	}

	// Initialize Redis.
	redisCache, err := cache.NewRedisCache()
	if err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}

	// Wire up service and handler layers.
	svc := service.NewShortenerService(db, redisCache)
	h := handler.NewHandler(svc, db, redisCache)

	// Configure router.
	r := mux.NewRouter()
	r.Use(middleware.Logging)
	r.Use(middleware.CORS)
	r.Use(middleware.RateLimit)
	h.RegisterRoutes(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
