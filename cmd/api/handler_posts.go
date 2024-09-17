package main

import (
	"net/http"

	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/data"
	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/database"
	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/validator"
	"github.com/google/uuid"
)

func (app *application) HandlerPostsGet(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title  string
		FeedID uuid.UUID
		data.Filters
	}

	user := app.contextGetUser(r)
	if user.IsAnonymous() {
		app.authenticationRequiredResponse(w, r)
		return
	}

	v := validator.New()

	qs := r.URL.Query()

	input.Title = app.readString(qs, "title", "")
	feedIDStr := app.readString(qs, "feed_id", "")
	if feedIDStr != "" {
		feedID, err := uuid.Parse(feedIDStr)
		if err != nil {
			v.AddError("feed_id", "must be a valid UUID")
		} else {
			input.FeedID = feedID
		}
	}

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)

	input.Filters.Sort = app.readString(qs, "sort", "published_at")
	input.Filters.SortSafelist = []string{"id", "title", "published_at", "-id", "-title", "-published_at"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	posts, err := app.db.GetPostsForUser(r.Context(), database.GetPostsForUserParams{
		UserID: user.ID,
		Title:  input.Title,
		FeedID: input.FeedID,
		Sort:   input.Filters.Sort,
		Lim:    int32(input.Filters.Limit()),  //#nosec G115
		Off:    int32(input.Filters.Offset()), //#nosec G115
	})
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	totalRecords := 0
	if len(posts) > 0 {
		totalRecords = int(posts[0].Count)
	}

	metadata := data.CalculateMetadata(totalRecords, input.Filters.Page, input.Filters.PageSize)

	err = app.writeJSON(w, http.StatusOK, envelope{
		"Metadata": metadata,
		"Posts":    data.DatabasePostsToPosts(posts),
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}
