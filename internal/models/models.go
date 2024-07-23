package models

import (
	"time"

	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/database"
	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `json:"name"`
}

type Feed struct {
	ID            uuid.UUID  `json:"id"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	Name          string     `json:"name"`
	Url           string     `json:"url"`
	UserID        uuid.UUID  `json:"userid"`
	LastFetchedAt *time.Time `json:"last_fetched_at"`
}

type FeedFollow struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	UserID    uuid.UUID `json:"userid"`
	FeedID    uuid.UUID `json:"feedid"`
}

type Post struct {
	ID          uuid.UUID `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Title       string    `json:"title"`
	Url         string    `json:"url"`
	Description string    `json:"description"`
	PublishedAt time.Time `json:"published_at"`
	FeedID      uuid.UUID `json:"feedid"`
}

func DatabaseUserToUser(user database.User) User {
	return User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Name:      user.Name,
	}
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

func DatabasePostToPost(post database.Post) Post {
	return Post{
		ID:          post.ID,
		CreatedAt:   post.CreatedAt,
		UpdatedAt:   post.UpdatedAt,
		Title:       post.Title,
		Url:         post.Url,
		Description: post.Description,
		PublishedAt: post.PublishedAt,
		FeedID:      post.FeedID,
	}
}

func DatabasePostsToPosts(posts []database.Post) []Post {
	result := make([]Post, len(posts))
	for i, post := range posts {
		result[i] = DatabasePostToPost(post)
	}
	return result
}
