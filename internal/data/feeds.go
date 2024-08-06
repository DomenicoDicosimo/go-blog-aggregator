package data

import (
	"time"

	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/database"
	"github.com/google/uuid"
)

type Feed struct {
	ID            uuid.UUID  `json:"id"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	Name          string     `json:"name"`
	Url           string     `json:"url"`
	UserID        uuid.UUID  `json:"userid"`
	LastFetchedAt *time.Time `json:"last_fetched_at"`
}

func DatabaseFeedToFeed(feed database.Feed) Feed {
	var lastFetchedAt *time.Time
	if feed.LastFetchedAt.Valid {
		lastFetchedAt = &feed.LastFetchedAt.Time
	}
	return Feed{
		ID:            feed.ID,
		CreatedAt:     feed.CreatedAt,
		UpdatedAt:     feed.UpdatedAt,
		LastFetchedAt: lastFetchedAt,
		Name:          feed.Name,
		Url:           feed.Url,
		UserID:        feed.UserID,
	}
}

func DatabaseFeedsToFeeds(feeds []database.Feed) []Feed {
	result := make([]Feed, len(feeds))
	for i, feed := range feeds {
		result[i] = DatabaseFeedToFeed(feed)
	}
	return result
}
