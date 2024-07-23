package handlers

import (
	"net/http"

	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/auth"
	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/database"
)

type authedHandler func(http.ResponseWriter, *http.Request, database.User)

func (cfg *APIConfig) MiddlewareAuth(handler authedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract API key
		apiKey, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Couldn't find apikey")
			return
		}

		// Verify API key and fetch user
		user, err := cfg.DB.GetUserByApiKey(r.Context(), apiKey)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Couldn't find user")
			return
		}

		// Call the next handler with the user
		handler(w, r, user)
	}
}