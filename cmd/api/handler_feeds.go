package main

import (
	"net/http"
	"time"

	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/data"
	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/database"
	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/validator"
	"github.com/google/uuid"
)

func (app *application) HandlerFeedsCreate(w http.ResponseWriter, r *http.Request, user database.User) {

	var input struct {
		Name string `json:"name" validate:"required,min=2,max=100"`
		URL  string `json:"url" validate:"required,url"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	v.ValidateStruct(input)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	feed, err := app.db.CreateFeed(r.Context(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      input.Name,
		Url:       input.URL,
		UserID:    user.ID,
	})
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	feedFollow, err := app.db.CreateFeedFollow(r.Context(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	response := struct {
		Feed       data.Feed       `json:"feed"`
		FeedFollow data.FeedFollow `json:"feed_follow"`
	}{
		Feed:       data.DatabaseFeedToFeed(feed),
		FeedFollow: data.DatabaseFeedFollowToFeedFollow(feedFollow),
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"Feed": response}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) HandlerFeedsGet(w http.ResponseWriter, r *http.Request) {
	feeds, err := app.db.GetFeeds(r.Context())
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"Feeds": data.DatabaseFeedsToFeeds(feeds)}, nil)
}
