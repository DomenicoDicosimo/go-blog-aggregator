package main

import (
	"net/http"
	"strconv"

	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/data"
	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/database"
)

func (app *application) HandlerPostsGet(w http.ResponseWriter, r *http.Request, user database.User) {

	limitString := r.URL.Query().Get("limit")
	if limitString == "" {
		limitString = "10"
	}
	limit, err := strconv.Atoi(limitString)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

	posts, err := app.db.GetPostsForUser(r.Context(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  int32(limit),
	})
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"Posts": data.DatabasePostsToPosts(posts)}, nil)
}
