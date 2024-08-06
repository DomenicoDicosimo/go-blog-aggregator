package main

import (
	"net/http"
	"strconv"

	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/data"
	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/database"
)

func (cfg *APIConfig) HandlerPostsGet(w http.ResponseWriter, r *http.Request, user database.User) {

	limitString := r.URL.Query().Get("limit")
	if limitString == "" {
		limitString = "10"
	}
	limit, err := strconv.Atoi(limitString)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Invalid limit value")
	}

	posts, err := cfg.DB.GetPostsByUser(r.Context(), database.GetPostsByUserParams{
		UserID: user.ID,
		Limit:  int32(limit),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't retrieve posts")
	}

	respondWithJSON(w, http.StatusOK, data.DatabasePostsToPosts(posts))
}
