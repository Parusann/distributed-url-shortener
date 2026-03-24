package model

import "time"

// URL represents a shortened URL record.
type URL struct {
	ID          int       `json:"id"`
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	ClickCount  int64     `json:"click_count"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// CreateURLRequest is the payload for creating a short URL.
type CreateURLRequest struct {
	URL       string `json:"url"`
	CustomCode string `json:"custom_code,omitempty"`
}

// URLResponse is the API response after creating a short URL.
type URLResponse struct {
	ShortCode   string `json:"short_code"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	CreatedAt   string `json:"created_at"`
}

// StatsResponse contains analytics for a short URL.
type StatsResponse struct {
	ShortCode   string `json:"short_code"`
	OriginalURL string `json:"original_url"`
	ClickCount  int64  `json:"click_count"`
	CreatedAt   string `json:"created_at"`
}

// HealthResponse is returned by the health check endpoint.
type HealthResponse struct {
	Status   string `json:"status"`
	Postgres string `json:"postgres"`
	Redis    string `json:"redis"`
}

// ErrorResponse is a standard error payload.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
