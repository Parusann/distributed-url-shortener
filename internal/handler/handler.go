package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/parus/distributed-url-shortener/internal/cache"
	"github.com/parus/distributed-url-shortener/internal/database"
	"github.com/parus/distributed-url-shortener/internal/model"
	"github.com/parus/distributed-url-shortener/internal/service"
)

// Handler holds dependencies for HTTP handlers.
type Handler struct {
	service *service.ShortenerService
	db      *database.PostgresStore
	cache   *cache.RedisCache
}

// NewHandler creates a Handler with all dependencies.
func NewHandler(svc *service.ShortenerService, db *database.PostgresStore, cache *cache.RedisCache) *Handler {
	return &Handler{service: svc, db: db, cache: cache}
}

// RegisterRoutes sets up all API routes on the given router.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/shorten", h.CreateShortURL).Methods("POST")
	r.HandleFunc("/api/urls/{code}/stats", h.GetURLStats).Methods("GET")
	r.HandleFunc("/health", h.HealthCheck).Methods("GET")
	r.HandleFunc("/{code}", h.RedirectURL).Methods("GET")
}

// CreateShortURL handles POST /api/shorten
func (h *Handler) CreateShortURL(w http.ResponseWriter, r *http.Request) {
	var req model.CreateURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	urlRecord, err := h.service.CreateShortURL(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidURL):
			respondError(w, http.StatusBadRequest, "Invalid URL. Must be a valid http/https URL.")
		case errors.Is(err, service.ErrCodeTaken):
			respondError(w, http.StatusConflict, "Custom short code is already in use.")
		default:
			log.Printf("Error creating short URL: %v", err)
			respondError(w, http.StatusInternalServerError, "Internal server error")
		}
		return
	}

	baseURL := getEnv("BASE_URL", "http://localhost:8080")
	resp := model.URLResponse{
		ShortCode:   urlRecord.ShortCode,
		ShortURL:    fmt.Sprintf("%s/%s", baseURL, urlRecord.ShortCode),
		OriginalURL: urlRecord.OriginalURL,
		CreatedAt:   urlRecord.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	respondJSON(w, http.StatusCreated, resp)
}

// RedirectURL handles GET /{code}
func (h *Handler) RedirectURL(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]

	originalURL, err := h.service.ResolveURL(r.Context(), code)
	if err != nil {
		if errors.Is(err, service.ErrURLNotFound) {
			respondError(w, http.StatusNotFound, "Short URL not found")
			return
		}
		log.Printf("Error resolving URL: %v", err)
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	http.Redirect(w, r, originalURL, http.StatusMovedPermanently)
}

// GetURLStats handles GET /api/urls/{code}/stats
func (h *Handler) GetURLStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]

	urlRecord, err := h.service.GetStats(code)
	if err != nil {
		if errors.Is(err, service.ErrURLNotFound) {
			respondError(w, http.StatusNotFound, "Short URL not found")
			return
		}
		log.Printf("Error getting stats: %v", err)
		respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	resp := model.StatsResponse{
		ShortCode:   urlRecord.ShortCode,
		OriginalURL: urlRecord.OriginalURL,
		ClickCount:  urlRecord.ClickCount,
		CreatedAt:   urlRecord.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	respondJSON(w, http.StatusOK, resp)
}

// HealthCheck handles GET /health
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	pgStatus := "up"
	if err := h.db.Ping(); err != nil {
		pgStatus = "down"
	}

	redisStatus := "up"
	if err := h.cache.Ping(context.Background()); err != nil {
		redisStatus = "down"
	}

	status := "healthy"
	code := http.StatusOK
	if pgStatus == "down" || redisStatus == "down" {
		status = "degraded"
		code = http.StatusServiceUnavailable
	}

	resp := model.HealthResponse{
		Status:   status,
		Postgres: pgStatus,
		Redis:    redisStatus,
	}
	respondJSON(w, code, resp)
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, model.ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
	})
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
