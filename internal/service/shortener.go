package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/url"
	"strings"

	"github.com/parus/distributed-url-shortener/internal/cache"
	"github.com/parus/distributed-url-shortener/internal/database"
	"github.com/parus/distributed-url-shortener/internal/model"
	"github.com/redis/go-redis/v9"
)

const (
	codeLength = 7
	charset    = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

var (
	ErrInvalidURL     = errors.New("invalid URL format")
	ErrCodeTaken      = errors.New("custom short code already taken")
	ErrURLNotFound    = errors.New("short URL not found")
	ErrCodeGeneration = errors.New("failed to generate unique short code")
)

// ShortenerService contains the business logic for URL shortening.
type ShortenerService struct {
	db    *database.PostgresStore
	cache *cache.RedisCache
}

// NewShortenerService creates a new service instance.
func NewShortenerService(db *database.PostgresStore, cache *cache.RedisCache) *ShortenerService {
	return &ShortenerService{db: db, cache: cache}
}

// CreateShortURL validates the URL and creates a short code.
func (s *ShortenerService) CreateShortURL(ctx context.Context, req model.CreateURLRequest) (*model.URL, error) {
	if err := validateURL(req.URL); err != nil {
		return nil, ErrInvalidURL
	}

	var shortCode string
	if req.CustomCode != "" {
		shortCode = req.CustomCode
		exists, err := s.db.ShortCodeExists(shortCode)
		if err != nil {
			return nil, fmt.Errorf("database error: %w", err)
		}
		if exists {
			return nil, ErrCodeTaken
		}
	} else {
		var err error
		shortCode, err = s.generateUniqueCode()
		if err != nil {
			return nil, err
		}
	}

	urlRecord, err := s.db.CreateURL(shortCode, req.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to create URL: %w", err)
	}

	// Cache the mapping for fast lookups.
	if err := s.cache.Set(ctx, shortCode, req.URL); err != nil {
		log.Printf("Warning: failed to cache URL %s: %v", shortCode, err)
	}

	return urlRecord, nil
}

// ResolveURL looks up the original URL, checking cache first.
func (s *ShortenerService) ResolveURL(ctx context.Context, shortCode string) (string, error) {
	// Check Redis cache first.
	originalURL, err := s.cache.Get(ctx, shortCode)
	if err == nil {
		// Cache hit — increment count asynchronously.
		go func() {
			if err := s.db.IncrementClickCount(shortCode); err != nil {
				log.Printf("Warning: failed to increment click count: %v", err)
			}
		}()
		return originalURL, nil
	}
	if !errors.Is(err, redis.Nil) {
		log.Printf("Warning: Redis error: %v", err)
	}

	// Cache miss — query PostgreSQL.
	urlRecord, err := s.db.GetURL(shortCode)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrURLNotFound
		}
		return "", fmt.Errorf("database error: %w", err)
	}

	// Populate cache for future requests.
	if err := s.cache.Set(ctx, shortCode, urlRecord.OriginalURL); err != nil {
		log.Printf("Warning: failed to cache URL %s: %v", shortCode, err)
	}

	// Increment click count.
	go func() {
		if err := s.db.IncrementClickCount(shortCode); err != nil {
			log.Printf("Warning: failed to increment click count: %v", err)
		}
	}()

	return urlRecord.OriginalURL, nil
}

// GetStats retrieves analytics for a short URL.
func (s *ShortenerService) GetStats(shortCode string) (*model.URL, error) {
	urlRecord, err := s.db.GetStats(shortCode)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrURLNotFound
		}
		return nil, fmt.Errorf("database error: %w", err)
	}
	return urlRecord, nil
}

func (s *ShortenerService) generateUniqueCode() (string, error) {
	for i := 0; i < 10; i++ {
		code := generateRandomCode(codeLength)
		exists, err := s.db.ShortCodeExists(code)
		if err != nil {
			return "", fmt.Errorf("database error: %w", err)
		}
		if !exists {
			return code, nil
		}
	}
	return "", ErrCodeGeneration
}

func generateRandomCode(length int) string {
	var sb strings.Builder
	charsetLen := big.NewInt(int64(len(charset)))
	for i := 0; i < length; i++ {
		n, _ := rand.Int(rand.Reader, charsetLen)
		sb.WriteByte(charset[n.Int64()])
	}
	return sb.String()
}

func validateURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("URL cannot be empty")
	}
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("URL must use http or https scheme")
	}
	if u.Host == "" {
		return fmt.Errorf("URL must have a host")
	}
	return nil
}
