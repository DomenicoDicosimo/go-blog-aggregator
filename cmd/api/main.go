package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/database"
	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/handlers"
	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/pkg/scraper"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load(".env")

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT environment variable is not set")
	}

	dbURL := os.Getenv("DB")
	if dbURL == "" {
		log.Fatal("DB environment variable is not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Problem connecting to database")
	}
	dbQueries := database.New(db)

	apiConfig := handlers.APIConfig{
		DB: dbQueries,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/healthz", handlers.HandlerReadiness)
	mux.HandleFunc("GET /v1/error", handlers.HandlerError)

	mux.HandleFunc("POST /v1/users", apiConfig.HandlerUsersCreate)
	mux.HandleFunc("GET /v1/users", apiConfig.MiddlewareAuth(apiConfig.HandlerUsersGet))

	mux.HandleFunc("POST /v1/feeds", apiConfig.MiddlewareAuth(apiConfig.HandlerFeedsCreate))
	mux.HandleFunc("GET /v1/feeds", apiConfig.HandlerFeedsGet)

	mux.HandleFunc("POST /v1/feed_follows", apiConfig.MiddlewareAuth(apiConfig.HandlerFeedFollowsCreate))
	mux.HandleFunc("DELETE /v1/feed_follows/{feedFollowID}", apiConfig.MiddlewareAuth(apiConfig.HandlerFeedFollowsDelete))
	mux.HandleFunc("GET /v1/feed_follows", apiConfig.MiddlewareAuth(apiConfig.HandlerFeedFollowsGet))

	mux.HandleFunc("GET /v1/posts", apiConfig.MiddlewareAuth(apiConfig.HandlerPostsGet))

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	const (
		collectionConcurrency = 10
		collectionInterval    = time.Minute
	)
	go scraper.StartScraping(dbQueries, collectionConcurrency, collectionInterval)

	log.Fatal(srv.ListenAndServe())
}
