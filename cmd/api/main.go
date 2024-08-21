package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/database"
	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/mailer"
	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/pkg/scraper"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type APIConfig struct {
	DB     *database.Queries
	Mailer mailer.Mailer
}

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

	mailerHost := os.Getenv("MAILER_HOST")
	if mailerHost == "" {
		log.Fatal("MAILER_HOST environment variable is not set")
	}

	mailerPortStr := os.Getenv("MAILER_PORT")
	if mailerPortStr == "" {
		log.Fatal("MAILER_PORT environment variable is not set")
	}

	mailerUsername := os.Getenv("MAILER_USERNAME")
	if mailerUsername == "" {
		log.Fatal("MAILER_USERNAME environment variable is not set")
	}

	mailerPassword := os.Getenv("MAILER_PASSWORD")
	if mailerPassword == "" {
		log.Fatal("MAILER_PASSWORD environment variable is not set")
	}

	mailerSender := os.Getenv("MAILER_SENDER")
	if mailerSender == "" {
		log.Fatal("MAILER_SENDER environment variable is not set")
	}

	mailerPort, err := strconv.Atoi(mailerPortStr)
	if err != nil {
		log.Fatal("Invalid MAILER_PORT: ", err)
	}

	mailerClient := mailer.New(mailerHost, mailerPort, mailerUsername, mailerPassword, mailerSender)

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Problem connecting to database")
	}
	dbQueries := database.New(db)

	apiConfig := APIConfig{
		DB:     dbQueries,
		Mailer: mailerClient,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/healthz", HandlerReadiness)
	mux.HandleFunc("GET /v1/error", HandlerError)

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
