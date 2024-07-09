package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	DB *database.Queries
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

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Problem connecting to database")
	}
	dbQueries := database.New(db)

	apiConfig := apiConfig{
		DB: dbQueries,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/healthz", handlerReadiness)
	mux.HandleFunc("GET /v1/error", handlerError)

	mux.HandleFunc("POST /v1/users", apiConfig.handlerUsersCreate)
	mux.HandleFunc("GET /v1/users", apiConfig.middlewareAuth(apiConfig.handlerUsersGet))

	mux.HandleFunc("POST /v1/feeds", apiConfig.middlewareAuth(apiConfig.handlerFeedsCreate))
	mux.HandleFunc("GET /v1/feeds", apiConfig.handlerFeedsGet)

	mux.HandleFunc("POST /v1/feed_follows", apiConfig.middlewareAuth(apiConfig.handlerFeedFollowsCreate))
	mux.HandleFunc("DELETE /v1/feed_follows/{feedFollowID}", apiConfig.middlewareAuth(apiConfig.handlerFeedFollowsDelete))
	mux.HandleFunc("GET /v1/feed_follows", apiConfig.middlewareAuth(apiConfig.handlerFeedFollowsGet))

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Fatal(srv.ListenAndServe())
}
