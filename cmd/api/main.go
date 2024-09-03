package main

import (
	"database/sql"
	"log"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/database"
	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/mailer"
	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/pkg/scraper"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type config struct {
	port    int
	env     string
	limiter struct {
		enabled bool
		rps     float64
		burst   int
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
}

type application struct {
	config config
	db     *database.Queries
	mailer mailer.Mailer
	logger *slog.Logger
	wg     sync.WaitGroup
}

func main() {
	var cfg config

	godotenv.Load(".env")

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

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
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer db.Close()

	dbQueries := database.New(db)

	app := &application{
		config: cfg,
		db:     dbQueries,
		mailer: mailerClient,
		logger: logger,
	}

	err = app.serve()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	const (
		collectionConcurrency = 10
		collectionInterval    = time.Minute
	)
	go scraper.StartScraping(dbQueries, collectionConcurrency, collectionInterval)
}
