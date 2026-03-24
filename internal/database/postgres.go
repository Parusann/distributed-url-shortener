package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/parus/distributed-url-shortener/internal/model"
)

// PostgresStore handles all database operations.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a connection pool and initializes the schema.
func NewPostgresStore() (*PostgresStore, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		getEnv("POSTGRES_HOST", "postgres"),
		getEnv("POSTGRES_PORT", "5432"),
		getEnv("POSTGRES_USER", "urlshortener"),
		getEnv("POSTGRES_PASSWORD", "urlshortener"),
		getEnv("POSTGRES_DB", "urlshortener"),
	)

	var db *sql.DB
	var err error

	// Retry connection with backoff for container startup ordering.
	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			err = db.Ping()
		}
		if err == nil {
			break
		}
		log.Printf("Waiting for PostgreSQL... attempt %d/10", i+1)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	store := &PostgresStore{db: db}
	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	log.Println("Connected to PostgreSQL")
	return store, nil
}

func (s *PostgresStore) migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS urls (
		id SERIAL PRIMARY KEY,
		short_code VARCHAR(20) UNIQUE NOT NULL,
		original_url TEXT NOT NULL,
		click_count BIGINT DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		expires_at TIMESTAMP WITH TIME ZONE
	);

	CREATE INDEX IF NOT EXISTS idx_urls_short_code ON urls(short_code);
	CREATE INDEX IF NOT EXISTS idx_urls_created_at ON urls(created_at);
	`
	_, err := s.db.Exec(query)
	return err
}

// CreateURL inserts a new shortened URL record.
func (s *PostgresStore) CreateURL(shortCode, originalURL string) (*model.URL, error) {
	url := &model.URL{}
	err := s.db.QueryRow(
		`INSERT INTO urls (short_code, original_url) VALUES ($1, $2)
		 RETURNING id, short_code, original_url, click_count, created_at`,
		shortCode, originalURL,
	).Scan(&url.ID, &url.ShortCode, &url.OriginalURL, &url.ClickCount, &url.CreatedAt)
	if err != nil {
		return nil, err
	}
	return url, nil
}

// GetURL retrieves a URL by its short code.
func (s *PostgresStore) GetURL(shortCode string) (*model.URL, error) {
	url := &model.URL{}
	err := s.db.QueryRow(
		`SELECT id, short_code, original_url, click_count, created_at FROM urls WHERE short_code = $1`,
		shortCode,
	).Scan(&url.ID, &url.ShortCode, &url.OriginalURL, &url.ClickCount, &url.CreatedAt)
	if err != nil {
		return nil, err
	}
	return url, nil
}

// IncrementClickCount atomically increments the click counter.
func (s *PostgresStore) IncrementClickCount(shortCode string) error {
	_, err := s.db.Exec(
		`UPDATE urls SET click_count = click_count + 1 WHERE short_code = $1`,
		shortCode,
	)
	return err
}

// GetStats returns URL info including click count.
func (s *PostgresStore) GetStats(shortCode string) (*model.URL, error) {
	return s.GetURL(shortCode)
}

// ShortCodeExists checks if a short code is already taken.
func (s *PostgresStore) ShortCodeExists(shortCode string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM urls WHERE short_code = $1)`,
		shortCode,
	).Scan(&exists)
	return exists, err
}

// Ping checks database connectivity.
func (s *PostgresStore) Ping() error {
	return s.db.Ping()
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
