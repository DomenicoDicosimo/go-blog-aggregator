package main

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

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
}

func (suite *APITestSuite) TearDownSuite() {
	err := suite.pgContainer.Terminate(suite.ctx)
	suite.NoError(err)
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

func (suite *APITestSuite) TestCreateAndGetFeed() {

	user, err := suite.queries.CreateUser(suite.ctx, database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      "Feed Owner",
	})
	require.NoError(suite.T(), err)

	feed, err := suite.queries.CreateFeed(suite.ctx, database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      "Test Feed",
		Url:       "http://example.com/feed",
		UserID:    user.ID,
	})
	require.NoError(suite.T(), err)
	require.NotEmpty(suite.T(), feed.ID)
	require.Equal(suite.T(), "Test Feed", feed.Name)

	feeds, err := suite.queries.GetFeeds(suite.ctx)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), feeds, 1)
	require.Equal(suite.T(), feed.ID, feeds[0].ID)
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
