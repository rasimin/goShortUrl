package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"goshorturl/entity"
	"goshorturl/proto"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/teris-io/shortid"
	"gorm.io/gorm"
)

// URLService provides methods for creating and getting URLs
type URLService struct {
	db    *gorm.DB
	redis *redis.Client
	proto.UnimplementedURLServiceServer
}

// NewURLService creates a new URLService
func NewURLService(db *gorm.DB, redis *redis.Client) *URLService {
	return &URLService{db: db, redis: redis}
}

// CreateURL creates a new URL and returns it
func (s *URLService) CreateURL(ctx context.Context, req *proto.CreateURLRequest) (*proto.CreateURLResponse, error) {
	shortURL, err := shortid.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate short URL: %w", err)
	}

	url := entity.URL{
		OriginalURL: req.OriginalUrl,
		ShortURL:    shortURL,
		CreatedAt:   time.Now(),
	}
	result := s.db.Create(&url)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to insert URL: %w", result.Error)
	}

	// Store the URL in Redis
	err = s.redis.Set(ctx, shortURL, req.OriginalUrl, 0).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to store URL in Redis: %w", err)
	}

	return &proto.CreateURLResponse{
		Url: &proto.URL{
			Id:          int32(url.ID),
			OriginalUrl: url.OriginalURL,
			ShortUrl:    url.ShortURL,
			CreatedAt:   url.CreatedAt.Format(time.RFC3339),
		},
	}, nil
}

// GetURL retrieves a URL by its short URL
func (s *URLService) GetURL(ctx context.Context, req *proto.GetURLRequest) (*proto.GetURLResponse, error) {
	// Check Redis first
	originalURL, err := s.redis.Get(ctx, req.ShortUrl).Result()
	if err == redis.Nil {
		// Not found in Redis, check the database
		var url entity.URL
		result := s.db.Where("short_url = ?", req.ShortUrl).First(&url)
		if result.Error != nil {
			return nil, fmt.Errorf("failed to find URL: %w", result.Error)
		}

		// Store the URL in Redis
		err = s.redis.Set(ctx, req.ShortUrl, url.OriginalURL, 0).Err()
		if err != nil {
			return nil, fmt.Errorf("failed to store URL in Redis: %w", err)
		}

		return &proto.GetURLResponse{
			Url: &proto.URL{
				Id:          int32(url.ID),
				OriginalUrl: url.OriginalURL,
				ShortUrl:    url.ShortURL,
				CreatedAt:   url.CreatedAt.Format(time.RFC3339),
			},
		}, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get URL from Redis: %w", err)
	}

	// Found in Redis
	return &proto.GetURLResponse{
		Url: &proto.URL{
			OriginalUrl: originalURL,
			ShortUrl:    req.ShortUrl,
		},
	}, nil
}

// GetShortURL retrieves a short URL by its long URL
func (s *URLService) GetShortURL(ctx context.Context, req *proto.GetShortURLRequest) (*proto.GetShortURLResponse, error) {
	var url entity.URL
	result := s.db.Where("original_url = ?", req.OriginalUrl).First(&url)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find URL: %w", result.Error)
	}

	return &proto.GetShortURLResponse{
		ShortUrl: url.ShortURL,
	}, nil
}

// RedirectURL handles redirecting to the original URL based on the short URL
func (s *URLService) RedirectURL(c *gin.Context) {
	shortURL := c.Param("shorturl")
	ctx := context.Background()
	originalURL, err := s.redis.Get(ctx, shortURL).Result()
	if err == redis.Nil {
		var url entity.URL
		result := s.db.Where("short_url = ?", shortURL).First(&url)
		if result.Error != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
			return
		}
		originalURL = url.OriginalURL
		// Store the URL in Redis
		s.redis.Set(ctx, shortURL, originalURL, 0)
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get URL from Redis"})
		return
	}
	c.Redirect(http.StatusFound, originalURL)
}
