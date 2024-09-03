package main

import (
	"net/http"
	"time"

	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/data"
	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/database"
	"github.com/google/uuid"
)

func (app *application) HandlerFeedFollowsCreate(w http.ResponseWriter, r *http.Request, user database.User) {
	var input struct {
		FeedID uuid.UUID `json:"feed_id"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	feed_follow, err := app.db.CreateFeedFollow(r.Context(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		UserID:    user.ID,
		FeedID:    input.FeedID,
	})
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"feed_follow": data.DatabaseFeedFollowToFeedFollow(feed_follow)}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) HandlerFeedFollowsDelete(w http.ResponseWriter, r *http.Request, user database.User) {

	feedIDString := r.PathValue("feedFollowID")
	feedID, err := uuid.Parse(feedIDString)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

	err = app.db.DeleteFeedFollow(r.Context(), feedID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"messege": "Feed follow deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) HandlerFeedFollowsGet(w http.ResponseWriter, r *http.Request, user database.User) {
	feedFollows, err := app.db.GetFeedFollows(r.Context(), user.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"feed follows": data.DatabaseFeedFollowsToFeedFollows(feedFollows)}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
