package data

import (
	"time"

	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/database"
	"github.com/google/uuid"
)

type FeedFollow struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	UserID    uuid.UUID `json:"userid"`
	FeedID    uuid.UUID `json:"feedid"`
}

func DatabaseFeedFollowToFeedFollow(feed_follow database.FeedFollow) FeedFollow {
	return FeedFollow{
		ID:        feed_follow.ID,
		CreatedAt: feed_follow.CreatedAt,
		UpdatedAt: feed_follow.UpdatedAt,
		UserID:    feed_follow.UserID,
		FeedID:    feed_follow.FeedID,
	}
}

func DatabaseFeedFollowsToFeedFollows(feedFollows []database.FeedFollow) []FeedFollow {
	result := make([]FeedFollow, len(feedFollows))
	for i, feedFollow := range feedFollows {
		result[i] = DatabaseFeedFollowToFeedFollow(feedFollow)
	}
	return result
}
