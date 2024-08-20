package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/data"
	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/database"
	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/validator"
	"github.com/google/uuid"
)

func (cfg *APIConfig) HandlerFeedsCreate(w http.ResponseWriter, r *http.Request, user database.User) {

	var input struct {
		Name string `json:"name" validate:"required,min=2,max=100"`
		URL  string `json:"url" validate:"required,url"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
		return
	}

	v := validator.New()
	v.ValidateStruct(input)
	if !v.Valid() {
		respondWithJSON(w, http.StatusUnprocessableEntity, v.Errors)
		return
	}

	feed, err := cfg.DB.CreateFeed(r.Context(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      input.Name,
		Url:       input.URL,
		UserID:    user.ID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could't create feed")
		return
	}

	feedFollow, err := cfg.DB.CreateFeedFollow(r.Context(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could't create ")
		return
	}

	response := struct {
		Feed       data.Feed       `json:"feed"`
		FeedFollow data.FeedFollow `json:"feed_follow"`
	}{
		Feed:       data.DatabaseFeedToFeed(feed),
		FeedFollow: data.DatabaseFeedFollowToFeedFollow(feedFollow),
	}

	respondWithJSON(w, http.StatusOK, response)
}

func (cfg *APIConfig) HandlerFeedsGet(w http.ResponseWriter, r *http.Request) {
	feeds, err := cfg.DB.GetFeeds(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get feeds")
	}

	respondWithJSON(w, http.StatusOK, data.DatabaseFeedsToFeeds(feeds))
}
