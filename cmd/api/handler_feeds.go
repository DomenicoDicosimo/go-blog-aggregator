package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/database"
	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/models"
	"github.com/google/uuid"
)

type feedParameters struct {
	Name string `json:"name" validate:"required,min=2,max=100"`
	URL  string `json:"url" validate:"required,url"`
}

func validateFeedParams(params feedParameters) error {
	return validate.Struct(params)
}

func (cfg *APIConfig) HandlerFeedsCreate(w http.ResponseWriter, r *http.Request, user database.User) {

	decoder := json.NewDecoder(r.Body)
	params := feedParameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
		return
	}

	err = validateFeedParams(params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid input parameters")
		return
	}

	feed, err := cfg.DB.CreateFeed(r.Context(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      params.Name,
		Url:       params.URL,
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
		Feed       models.Feed       `json:"feed"`
		FeedFollow models.FeedFollow `json:"feed_follow"`
	}{
		Feed:       models.DatabaseFeedToFeed(feed),
		FeedFollow: models.DatabaseFeedFollowToFeedFollow(feedFollow),
	}

	respondWithJSON(w, http.StatusOK, response)
}

func (cfg *APIConfig) HandlerFeedsGet(w http.ResponseWriter, r *http.Request) {
	feeds, err := cfg.DB.GetFeeds(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get feeds")
	}

	respondWithJSON(w, http.StatusOK, models.DatabaseFeedsToFeeds(feeds))
}
