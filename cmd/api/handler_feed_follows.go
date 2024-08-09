package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/database"
	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/data"
	"github.com/google/uuid"
)

func (cfg *APIConfig) HandlerFeedFollowsCreate(w http.ResponseWriter, r *http.Request, user database.User) {
	type parameters struct {
		FeedID uuid.UUID `json:"feed_id"`
	}


	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
		return
	}

	feed_follow, err := cfg.DB.CreateFeedFollow(r.Context(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		UserID:    user.ID,
		FeedID:    params.FeedID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could't create ")
		return
	}

	respondWithJSON(w, http.StatusOK, models.DatabaseFeedFollowToFeedFollow(feed_follow))
}

func (cfg *APIConfig) HandlerFeedFollowsDelete(w http.ResponseWriter, r *http.Request, user database.User) {

	feedIDString := r.PathValue("feedFollowID")
	feedID, err := uuid.Parse(feedIDString)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Invalid feed id")
	}

	err = cfg.DB.DeleteFeedFollow(r.Context(), feedID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could't create user")
		return
	}

	respondWithJSON(w, http.StatusOK, "Feed follow deleted")
}

func (cfg *APIConfig) HandlerFeedFollowsGet(w http.ResponseWriter, r *http.Request, user database.User) {
	feedFollows, err := cfg.DB.GetFeedFollows(r.Context(), user.ID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get feeds")
	}

	respondWithJSON(w, http.StatusOK, models.DatabaseFeedFollowsToFeedFollows(feedFollows))
}
