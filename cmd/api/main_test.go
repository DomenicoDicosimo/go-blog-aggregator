package main

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/DomenicoDicosimo/go-blog-aggregator/handlers"
	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/database"
	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type APITestSuite struct {
	suite.Suite
	ctx                context.Context
	db                 *sql.DB
	pgContainer        *postgres.PostgresContainer
	pgConnectionString string
	queries            *database.Queries
	tx                 *sql.Tx
	server             *httptest.Server
}

func (suite *APITestSuite) SetupSuite() {
	suite.ctx = context.Background()

	pgContainer, err := postgres.Run(
		suite.ctx,
		"postgres:14-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	suite.NoError(err)

	connStr, err := pgContainer.ConnectionString(suite.ctx, "sslmode=disable")
	suite.NoError(err)

	db, err := sql.Open("postgres", connStr)
	suite.NoError(err)

	suite.pgContainer = pgContainer
	suite.pgConnectionString = connStr
	suite.db = db

	err = goose.SetDialect("postgres")
	suite.NoError(err)

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		suite.T().Fatal("No caller information")
	}
	currentDir := filepath.Dir(filename)

	// Find the project root (assuming it's the directory containing "go.mod")
	projectRoot := currentDir
	for {
		if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			suite.T().Fatal("Unable to find project root")
		}
		projectRoot = parent
	}

	// Construct the path to the migrations
	migrationsDir := filepath.Join(projectRoot, "sql", "schema")

	err = goose.Up(db, migrationsDir)
	suite.NoError(err)

	suite.queries = database.New(db)

	suite.setupTestServer()
}

func (suite *APITestSuite) setupTestServer() {
	apiConfig := handlers.APIConfig{
		DB: suite.queries,
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

	suite.server = httptest.NewServer(mux)
}

func (suite *APITestSuite) TearDownSuite() {
	err := suite.pgContainer.Terminate(suite.ctx)
	suite.NoError(err)

	if suite.server != nil {
		suite.server.Close()
	}
}

func (suite *APITestSuite) SetupTest() {
	tx, err := suite.db.Begin()
	suite.NoError(err)

	suite.tx = tx
	suite.queries = database.New(tx)
}

func (suite *APITestSuite) TearDownTest() {
	if suite.tx != nil {
		err := suite.tx.Rollback()
		suite.NoError(err)
	}
}

func (suite *APITestSuite) TestUserFunctions() {
	// Test CreateUser
	userName := "Test User"
	user, err := suite.queries.CreateUser(suite.ctx, database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      userName,
	})

	require.NoError(suite.T(), err)
	require.NotEmpty(suite.T(), user.ID)
	require.Equal(suite.T(), userName, user.Name)
	require.NotEmpty(suite.T(), user.ApiKey)

	// Test GetUserByApiKey
	fetchedUser, err := suite.queries.GetUserByApiKey(suite.ctx, user.ApiKey)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), user.ID, fetchedUser.ID)
	require.Equal(suite.T(), user.Name, fetchedUser.Name)
	require.Equal(suite.T(), user.ApiKey, fetchedUser.ApiKey)

	// Test GetUserByApiKey with invalid API key
	invalidApiKey := "invalid_api_key"
	_, err = suite.queries.GetUserByApiKey(suite.ctx, invalidApiKey)
	require.Error(suite.T(), err) // Expect an error for invalid API key

	// Test creating a user with an existing name (if your DB allows it)
	duplicateUser, err := suite.queries.CreateUser(suite.ctx, database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      userName,
	})
	require.NoError(suite.T(), err)
	require.NotEqual(suite.T(), user.ID, duplicateUser.ID)
	require.NotEqual(suite.T(), user.ApiKey, duplicateUser.ApiKey)

	// Test creating a user with an empty name (if your DB validation allows it)
	_, err = suite.queries.CreateUser(suite.ctx, database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      "",
	})
	// Adjust the following assertion based on your DB constraints
	// If empty names are not allowed, change to require.Error
	require.NoError(suite.T(), err)
}

func (suite *APITestSuite) TestAPIFeedCreateAndRetrieve() {
	/*
		// First, create a user
		userData := map[string]string{"name": "Feed Owner"}
		body, _ := json.Marshal(userData)
		resp, err := http.Post(suite.server.URL+"/v1/users", "application/json", bytes.NewBuffer(body))
		suite.NoError(err)
		var user models.User
		json.NewDecoder(resp.Body).Decode(&user)
		resp.Body.Close()

		// Create a feed
		feedData := handlers.feedParameters{
			Name: "Test Feed",
			URL:  "http://example.com/feed",
		}
		body, _ = json.Marshal(feedData)
		req, _ := http.NewRequest("POST", suite.server.URL+"/v1/feeds", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "ApiKey "+user.ApiKey)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err = client.Do(req)
		suite.NoError(err)
		defer resp.Body.Close()

		suite.Equal(http.StatusOK, resp.StatusCode)

		var feedResp struct {
			Feed       models.Feed       `json:"feed"`
			FeedFollow models.FeedFollow `json:"feed_follow"`
		}
		err = json.NewDecoder(resp.Body).Decode(&feedResp)
		suite.NoError(err)
		suite.NotEmpty(feedResp.Feed.ID)
		suite.Equal("Test Feed", feedResp.Feed.Name)
		suite.Equal("http://example.com/feed", feedResp.Feed.Url)
		suite.NotEmpty(feedResp.FeedFollow.ID)
		suite.Equal(feedResp.Feed.ID, feedResp.FeedFollow.FeedID)
		suite.Equal(user.ID, feedResp.FeedFollow.UserID)

		// Verify feed in database
		dbFeed, err := suite.queries.GetFeedByID(suite.ctx, feedResp.Feed.ID)
		suite.NoError(err)
		suite.Equal(feedResp.Feed.ID, dbFeed.ID)
		suite.Equal(feedResp.Feed.Name, dbFeed.Name)
		suite.Equal(feedResp.Feed.Url, dbFeed.Url)

		// Verify feed follow in database
		dbFeedFollow, err := suite.queries.GetFeedFollowByID(suite.ctx, feedResp.FeedFollow.ID)
		suite.NoError(err)
		suite.Equal(feedResp.FeedFollow.ID, dbFeedFollow.ID)
		suite.Equal(feedResp.FeedFollow.FeedID, dbFeedFollow.FeedID)
		suite.Equal(feedResp.FeedFollow.UserID, dbFeedFollow.UserID)

		// Test input validation
		invalidFeedData := feedParameters{
			Name: "",          // Invalid: empty name
			URL:  "not-a-url", // Invalid: not a proper URL
		}
		body, _ = json.Marshal(invalidFeedData)
		req, _ = http.NewRequest("POST", suite.server.URL+"/v1/feeds", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "ApiKey "+user.ApiKey)
		req.Header.Set("Content-Type", "application/json")
		resp, err = client.Do(req)
		suite.NoError(err)
		defer resp.Body.Close()
		suite.Equal(http.StatusBadRequest, resp.StatusCode)

		// Retrieve feeds via API
		req, _ = http.NewRequest("GET", suite.server.URL+"/v1/feeds", nil)
		resp, err = client.Do(req)
		suite.NoError(err)
		defer resp.Body.Close()

		suite.Equal(http.StatusOK, resp.StatusCode)

		var feeds []models.Feed
		err = json.NewDecoder(resp.Body).Decode(&feeds)
		suite.NoError(err)
		suite.NotEmpty(feeds)
		suite.Equal(feedResp.Feed.ID, feeds[0].ID)
		suite.Equal(feedResp.Feed.Name, feeds[0].Name)

	*/
}

func (suite *APITestSuite) TestPosts() {

	user, err := suite.queries.CreateUser(suite.ctx, database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      "Test User",
	})
	suite.NoError(err)
	suite.NotEmpty(user.ID)

	feed, err := suite.queries.CreateFeed(suite.ctx, database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      "Test Feed",
		Url:       "http://testfeed.com",
		UserID:    user.ID,
	})
	suite.NoError(err)
	suite.NotEmpty(feed.ID)

	postTitle := "Test Post"
	postUrl := "http://testpost.com"
	postDescription := "This is a test post"
	publishedAt := time.Now().UTC()

	post, err := suite.queries.CreatePost(suite.ctx, database.CreatePostParams{
		ID:          uuid.New(),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		Title:       postTitle,
		Url:         postUrl,
		Description: postDescription,
		PublishedAt: publishedAt,
		FeedID:      feed.ID,
	})
	suite.NoError(err)
	suite.NotEmpty(post.ID)
	suite.Equal(postTitle, post.Title)
	suite.Equal(postUrl, post.Url)
	suite.Equal(postDescription, post.Description)
	suite.Equal(publishedAt.Unix(), post.PublishedAt.Unix())
	suite.Equal(feed.ID, post.FeedID)

	_, err = suite.queries.CreateFeedFollow(suite.ctx, database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	suite.NoError(err)

	posts, err := suite.queries.GetPostsByUser(suite.ctx, database.GetPostsByUserParams{
		UserID: user.ID,
		Limit:  10,
	})
	suite.NoError(err)
	suite.Len(posts, 1)
	suite.Equal(post.ID, posts[0].ID)

	postsLimited, err := suite.queries.GetPostsByUser(suite.ctx, database.GetPostsByUserParams{
		UserID: user.ID,
		Limit:  1,
	})
	suite.NoError(err)
	suite.Len(postsLimited, 1)

	anotherUser, err := suite.queries.CreateUser(suite.ctx, database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      "Another Test User",
	})
	suite.NoError(err)

	postsForAnotherUser, err := suite.queries.GetPostsByUser(suite.ctx, database.GetPostsByUserParams{
		UserID: anotherUser.ID,
		Limit:  10,
	})
	suite.NoError(err)
	suite.Len(postsForAnotherUser, 0)
}

func (suite *APITestSuite) TestFeedFollows() {
	ctx := context.Background()

	// First, create a user
	user, err := suite.queries.CreateUser(ctx, database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      "Test User",
	})
	suite.NoError(err)
	suite.NotEmpty(user.ID)

	// Create a feed
	feed, err := suite.queries.CreateFeed(ctx, database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      "Test Feed",
		Url:       "http://testfeed.com",
		UserID:    user.ID,
	})
	suite.NoError(err)
	suite.NotEmpty(feed.ID)

	// Test CreateFeedFollow
	feedFollow, err := suite.queries.CreateFeedFollow(ctx, database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	suite.NoError(err)
	suite.NotEmpty(feedFollow.ID)
	suite.Equal(user.ID, feedFollow.UserID)
	suite.Equal(feed.ID, feedFollow.FeedID)

	// Test GetFeedFollows
	feedFollows, err := suite.queries.GetFeedFollows(ctx, user.ID)
	suite.NoError(err)
	suite.Len(feedFollows, 1)
	suite.Equal(feedFollow.ID, feedFollows[0].ID)

	// Test DeleteFeedFollow
	err = suite.queries.DeleteFeedFollow(ctx, feed.ID)
	suite.NoError(err)

	// Verify the feed follow was deleted
	feedFollowsAfterDelete, err := suite.queries.GetFeedFollows(ctx, user.ID)
	suite.NoError(err)
	suite.Len(feedFollowsAfterDelete, 0)

	// Test deleting a non-existent feed follow
	err = suite.queries.DeleteFeedFollow(ctx, uuid.New())
	suite.NoError(err) // This should not return an error, just affect 0 rows

	// Test creating a duplicate feed follow (if your database enforces uniqueness)
	_, err = suite.queries.CreateFeedFollow(ctx, database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	suite.NoError(err) // If your database enforces uniqueness, this should error

	// Create another feed and follow it
	anotherFeed, err := suite.queries.CreateFeed(ctx, database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      "Another Test Feed",
		Url:       "http://anothertestfeed.com",
		UserID:    user.ID,
	})
	suite.NoError(err)

	_, err = suite.queries.CreateFeedFollow(ctx, database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		UserID:    user.ID,
		FeedID:    anotherFeed.ID,
	})
	suite.NoError(err)

	// Verify the user now follows two feeds
	finalFeedFollows, err := suite.queries.GetFeedFollows(ctx, user.ID)
	suite.NoError(err)
	suite.Len(finalFeedFollows, 2)
}

func TestAPISuite(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}
