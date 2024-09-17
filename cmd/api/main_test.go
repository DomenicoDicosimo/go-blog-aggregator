package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/database"
	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/mailer"
	"github.com/google/uuid"
	smtpmock "github.com/mocktools/go-smtp-mock/v2"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type APITestSuite struct {
	suite.Suite
	ctx                    context.Context
	db                     *sql.DB
	pgContainer            *postgres.PostgresContainer
	pgConnectionString     string
	queries                *database.Queries
	tx                     *sql.Tx
	server                 *httptest.Server
	app                    *application
	smtpServer             *smtpmock.Server
	authenticatedClient    *http.Client
	authenticatedUserEmail string
}

type MailtrapEmail struct {
	ID        int    `json:"id"`
	Subject   string `json:"subject"`
	SentAt    string `json:"sent_at"`
	FromEmail string `json:"from_email"`
	FromName  string `json:"from_name"`
	ToEmail   string `json:"to_email"`
	ToName    string `json:"to_name"`
	HTMLBody  string `json:"html_body"`
	TextBody  string `json:"text_body"`
}

func (suite *APITestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Setup PostgreSQL container
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
	suite.Require().NoError(err)

	connStr, err := pgContainer.ConnectionString(suite.ctx, "sslmode=disable")
	suite.Require().NoError(err)

	db, err := sql.Open("postgres", connStr)
	suite.Require().NoError(err)

	suite.pgContainer = pgContainer
	suite.pgConnectionString = connStr
	suite.db = db

	// Run migrations
	err = goose.SetDialect("postgres")
	suite.Require().NoError(err)

	_, filename, _, ok := runtime.Caller(0)
	suite.Require().True(ok, "No caller information")

	projectRoot := filepath.Dir(filename)
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

	migrationsDir := filepath.Join(projectRoot, "sql", "schema")
	err = goose.Up(db, migrationsDir)
	suite.Require().NoError(err)

	suite.queries = database.New(db)

	// Setup application
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg := config{
		port: 8080,
		env:  "testing",
		limiter: struct {
			enabled bool
			rps     float64
			burst   int
		}{
			enabled: false,
			rps:     100,
			burst:   100,
		},
		cors: struct {
			trustedOrigins []string
		}{
			trustedOrigins: []string{""},
		},
	}

	suite.smtpServer = smtpmock.New(smtpmock.ConfigurationAttr{
		LogToStdout:       true,
		LogServerActivity: true,
	})

	err = suite.smtpServer.Start()
	suite.Require().NoError(err)

	smtpPort := suite.smtpServer.PortNumber()

	mailerClient := mailer.New(
		"localhost",
		smtpPort,
		"username", // These credentials won't be checked by the mock server
		"password",
		"Go-Blog-Aggregator <no-reply@goblogagg.com>",
	)

	suite.app = &application{
		config: cfg,
		db:     suite.queries,
		mailer: mailerClient,
		logger: logger,
	}

	suite.setupTestServer()

	suite.SetupAuthenticatedClient()
}

func (suite *APITestSuite) setupTestServer() {
	// Create a new http.Server using the application's routes
	srv := &http.Server{
		Handler: suite.app.routes(),
	}

	// Create a test server
	suite.server = httptest.NewServer(srv.Handler)

	// Parse the test server URL
	serverURL, err := url.Parse(suite.server.URL)
	suite.Require().NoError(err)

	// Update the app config with the test server's port
	suite.app.config.port, err = strconv.Atoi(serverURL.Port())
	suite.Require().NoError(err)

	// Log the test server information
	suite.app.logger.Info("started test server", "url", suite.server.URL)
}

func (suite *APITestSuite) TearDownSuite() {
	// Close the test server
	if suite.server != nil {
		suite.server.Close()
		suite.app.logger.Info("stopped test server")
	}

	if suite.smtpServer != nil {
		suite.smtpServer.Stop()
	}

	// Close the database connection
	if suite.db != nil {
		err := suite.db.Close()
		suite.Require().NoError(err)
	}

	// Terminate the PostgreSQL container
	if suite.pgContainer != nil {
		err := suite.pgContainer.Terminate(suite.ctx)
		suite.Require().NoError(err)
	}

	// Wait for any background tasks to complete
	suite.app.wg.Wait()
}

func (suite *APITestSuite) SetupTest() {
	// Start a new transaction
	tx, err := suite.db.Begin()
	suite.Require().NoError(err)

	// Replace the application's database queries with a new one using the transaction
	suite.app.db = database.New(tx)

	// Store the transaction for later use
	suite.tx = tx
}

func (suite *APITestSuite) TearDownTest() {
	// Rollback the transaction to reset the database state
	if suite.tx != nil {
		err := suite.tx.Rollback()
		suite.Require().NoError(err)
	}

	// Reset the application's database queries to use the main database connection
	suite.app.db = database.New(suite.db)
}

func (suite *APITestSuite) SetupAuthenticatedClient() {
	email, authToken, err := suite.CreateActivatedUser()
	suite.Require().NoError(err, "Failed to create and activate user")

	suite.T().Logf("Created and activated user with email: %s", email)

	// Create a new http.Client with a custom Transport
	client := &http.Client{
		Transport: &AuthenticatedTransport{
			Base:      http.DefaultTransport,
			AuthToken: authToken,
		},
	}

	// Store the authenticated client and user email in the suite
	suite.authenticatedClient = client
	suite.authenticatedUserEmail = email
}

// AuthenticatedTransport is a custom http.RoundTripper that adds the authentication token to requests
type AuthenticatedTransport struct {
	Base      http.RoundTripper
	AuthToken string
}

func (t *AuthenticatedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", "Bearer "+t.AuthToken)
	return t.Base.RoundTrip(req)
}

func (suite *APITestSuite) CreateActivatedUser() (email, authToken string, err error) {
	// Generate a unique email
	email = fmt.Sprintf("test%d@example.com", time.Now().UnixNano())
	name := "Test User"
	password := "password123"

	// Send POST request to create user
	resp, err := suite.server.Client().Post(
		suite.server.URL+"/v1/users",
		"application/json",
		strings.NewReader(fmt.Sprintf(`{"name":"%s","email":"%s","password":"%s"}`, name, email, password)),
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to create user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("failed to create user, status: %d, body: %s", resp.StatusCode, body)
	}

	// Wait for the email to be "sent"
	time.Sleep(1 * time.Second)

	// Check if an email was "sent" using the mock SMTP server
	messages := suite.smtpServer.Messages()
	if len(messages) != 1 {
		return "", "", fmt.Errorf("expected 1 email to be sent, got %d", len(messages))
	}

	sentEmail := messages[0]
	emailContent := sentEmail.MsgRequest()

	// Extract activation token from the email content
	tokenRegex := regexp.MustCompile(`"token": "(\w+)"`)
	matches := tokenRegex.FindStringSubmatch(emailContent)
	if len(matches) != 2 {
		return "", "", fmt.Errorf("activation token not found in email")
	}
	activationToken := matches[1]

	// Send PUT request to activate user
	req, err := http.NewRequest(http.MethodPut, suite.server.URL+"/v1/users/activated", strings.NewReader(fmt.Sprintf(`{"token":"%s"}`, activationToken)))
	if err != nil {
		return "", "", fmt.Errorf("failed to create activation request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err = suite.server.Client().Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to send activation request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("failed to activate user, status: %d, body: %s", resp.StatusCode, body)
	}

	// Send POST request to create authentication token
	authResp, err := suite.server.Client().Post(
		suite.server.URL+"/v1/tokens/authentication",
		"application/json",
		strings.NewReader(fmt.Sprintf(`{"email":"%s","password":"%s"}`, email, password)),
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to create authentication token: %w", err)
	}
	defer authResp.Body.Close()

	if authResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(authResp.Body)
		return "", "", fmt.Errorf("failed to create authentication token, status: %d, body: %s", authResp.StatusCode, body)
	}

	// Read and parse the authentication token from the response
	var authTokenResponse struct {
		AuthenticationToken struct {
			Token string `json:"token"`
		} `json:"authentication_token"`
	}
	err = json.NewDecoder(authResp.Body).Decode(&authTokenResponse)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode authentication token response: %w", err)
	}

	return email, authTokenResponse.AuthenticationToken.Token, nil
}

func (suite *APITestSuite) TestUserAuth() {
	// Verify user is activated in the database
	user, err := suite.app.db.GetUserByEmail(context.Background(), suite.authenticatedUserEmail)
	suite.Require().NoError(err)
	suite.Require().True(user.Activated, "User is not activated in the database")

	// Test using the authenticated client
	resp, err := suite.authenticatedClient.Get(suite.server.URL + "/v1/feeds")
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Require().Equal(http.StatusOK, resp.StatusCode, "Failed to access protected route with auth token")
}

func (suite *APITestSuite) TestFeeds() {
	// Test creating a feed
	createFeedBody := `{"name":"Test Feed","url":"http://example.com/rss/feed.xml"}`
	resp, err := suite.authenticatedClient.Post(suite.server.URL+"/v1/feeds", "application/json", strings.NewReader(createFeedBody))
	suite.Require().NoError(err)
	defer resp.Body.Close()

	// Log the full response for debugging
	respBody, err := io.ReadAll(resp.Body)
	suite.Require().NoError(err)
	suite.T().Logf("Create feed response: %s", string(respBody))

	// Reset the response body for further reading
	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))

	suite.Require().Equal(http.StatusOK, resp.StatusCode, "Failed to create feed")

	var createFeedResponse struct {
		Feed struct {
			Feed struct {
				ID            uuid.UUID  `json:"id"`
				CreatedAt     time.Time  `json:"created_at"`
				UpdatedAt     time.Time  `json:"updated_at"`
				Name          string     `json:"name"`
				URL           string     `json:"url"`
				UserID        uuid.UUID  `json:"userid"`
				LastFetchedAt *time.Time `json:"last_fetched_at"`
			} `json:"feed"`
			FeedFollow struct {
				ID        uuid.UUID `json:"id"`
				CreatedAt time.Time `json:"created_at"`
				UpdatedAt time.Time `json:"updated_at"`
				UserID    uuid.UUID `json:"userid"`
				FeedID    uuid.UUID `json:"feedid"`
			} `json:"feed_follow"`
		} `json:"Feed"`
	}
	err = json.NewDecoder(resp.Body).Decode(&createFeedResponse)
	suite.Require().NoError(err)

	// Log the parsed response
	suite.T().Logf("Parsed create feed response: %+v", createFeedResponse)

	suite.Require().NotEqual(uuid.Nil, createFeedResponse.Feed.Feed.ID, "Feed ID should not be empty")
	suite.Require().Equal("Test Feed", createFeedResponse.Feed.Feed.Name)
	suite.Require().Equal("http://example.com/rss/feed.xml", createFeedResponse.Feed.Feed.URL)

	// Test getting all feeds
	resp, err = suite.authenticatedClient.Get(suite.server.URL + "/v1/feeds")
	suite.Require().NoError(err)
	defer resp.Body.Close()
	suite.Require().Equal(http.StatusOK, resp.StatusCode, "Failed to get feeds")

	var getFeedsResponse struct {
		Feeds []struct {
			ID   uuid.UUID `json:"id"`
			Name string    `json:"name"`
			URL  string    `json:"url"`
		} `json:"Feeds"`
	}
	err = json.NewDecoder(resp.Body).Decode(&getFeedsResponse)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(getFeedsResponse.Feeds, "Feeds list should not be empty")

	// Check if the created feed is in the list of all feeds
	var foundCreatedFeed bool
	for _, feed := range getFeedsResponse.Feeds {
		if feed.ID == createFeedResponse.Feed.Feed.ID {
			foundCreatedFeed = true
			suite.Require().Equal(createFeedResponse.Feed.Feed.Name, feed.Name)
			suite.Require().Equal(createFeedResponse.Feed.Feed.URL, feed.URL)
			break
		}
	}
	suite.Require().True(foundCreatedFeed, "Created feed should be in the list of all feeds")
}

func (suite *APITestSuite) TestFeedFollows() {

	// First, create a feed (which automatically creates a feed follow)
	createFeedBody := `{"name":"Test Feed for Follow","url":"http://example.com/rss/feed2.xml"}`
	resp, err := suite.authenticatedClient.Post(suite.server.URL+"/v1/feeds", "application/json", strings.NewReader(createFeedBody))
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Require().Equal(http.StatusOK, resp.StatusCode, "Failed to create feed")

	var createFeedResponse struct {
		Feed struct {
			Feed struct {
				ID            uuid.UUID  `json:"id"`
				CreatedAt     time.Time  `json:"created_at"`
				UpdatedAt     time.Time  `json:"updated_at"`
				Name          string     `json:"name"`
				URL           string     `json:"url"`
				UserID        uuid.UUID  `json:"userid"`
				LastFetchedAt *time.Time `json:"last_fetched_at"`
			} `json:"feed"`
			FeedFollow struct {
				ID        uuid.UUID `json:"id"`
				CreatedAt time.Time `json:"created_at"`
				UpdatedAt time.Time `json:"updated_at"`
				UserID    uuid.UUID `json:"userid"`
				FeedID    uuid.UUID `json:"feedid"`
			} `json:"feed_follow"`
		} `json:"Feed"`
	}
	err = json.NewDecoder(resp.Body).Decode(&createFeedResponse)
	suite.Require().NoError(err)

	suite.T().Logf("Created feed with ID: %s", createFeedResponse.Feed.Feed.ID)
	suite.T().Logf("Automatically created feed follow with ID: %s", createFeedResponse.Feed.FeedFollow.ID)

	// Test deleting the automatically created feed follow
	deleteURL := fmt.Sprintf("%s/v1/feed_follows/%s", suite.server.URL, createFeedResponse.Feed.FeedFollow.ID)
	suite.T().Logf("Attempting to delete feed follow at URL: %s", deleteURL)
	req, err := http.NewRequest(http.MethodDelete, deleteURL, nil)
	suite.Require().NoError(err)
	resp, err = suite.authenticatedClient.Do(req)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	// Log the response body for debugging
	body, _ := io.ReadAll(resp.Body)
	suite.T().Logf("Delete feed follow response: %s", string(body))

	suite.Require().Equal(http.StatusOK, resp.StatusCode, "Failed to delete feed follow")

	var deleteResponse struct {
		Message string `json:"messege"`
	}
	err = json.Unmarshal(body, &deleteResponse)
	suite.Require().NoError(err)
	suite.Require().Equal("Feed follow deleted", deleteResponse.Message)

	// Test creating a new feed follow
	createFollowBody := fmt.Sprintf(`{"feed_id":"%s"}`, createFeedResponse.Feed.Feed.ID)
	resp, err = suite.authenticatedClient.Post(suite.server.URL+"/v1/feed_follows", "application/json", strings.NewReader(createFollowBody))
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Require().Equal(http.StatusOK, resp.StatusCode, "Failed to create feed follow")

	var createFollowResponse struct {
		FeedFollow struct {
			ID     uuid.UUID `json:"id"`
			FeedID uuid.UUID `json:"feedid"`
		} `json:"feed_follow"`
	}
	err = json.NewDecoder(resp.Body).Decode(&createFollowResponse)
	suite.Require().NoError(err)

	suite.Require().NotEqual(uuid.Nil, createFollowResponse.FeedFollow.ID, "Feed follow ID should not be empty")
	suite.Require().Equal(createFeedResponse.Feed.Feed.ID, createFollowResponse.FeedFollow.FeedID)

	// Test getting all feed follows
	resp, err = suite.authenticatedClient.Get(suite.server.URL + "/v1/feed_follows")
	suite.Require().NoError(err)
	defer resp.Body.Close()
	suite.Require().Equal(http.StatusOK, resp.StatusCode, "Failed to get feed follows")

	var getFollowsResponse struct {
		FeedFollows []struct {
			ID     uuid.UUID `json:"id"`
			FeedID uuid.UUID `json:"feedid"`
		} `json:"feed follows"`
	}
	err = json.NewDecoder(resp.Body).Decode(&getFollowsResponse)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(getFollowsResponse.FeedFollows, "Feed follows list should not be empty")

	// Check if the newly created feed follow is in the list
	found := false
	for _, follow := range getFollowsResponse.FeedFollows {
		if follow.ID == createFollowResponse.FeedFollow.ID {
			found = true
			break
		}
	}
	suite.Require().True(found, "Newly created feed follow should be in the list")

	// Test deleting the newly created feed follow
	deleteURL = fmt.Sprintf("%s/v1/feed_follows/%s", suite.server.URL, createFollowResponse.FeedFollow.ID)
	suite.T().Logf("Attempting to delete feed follow at URL: %s", deleteURL)
	req, err = http.NewRequest(http.MethodDelete, deleteURL, nil)
	suite.Require().NoError(err)
	resp, err = suite.authenticatedClient.Do(req)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	// Log the response body for debugging
	body, _ = io.ReadAll(resp.Body)
	suite.T().Logf("Delete feed follow response: %s", string(body))

	suite.Require().Equal(http.StatusOK, resp.StatusCode, "Failed to delete feed follow")

	err = json.Unmarshal(body, &deleteResponse)
	suite.Require().NoError(err)
	suite.Require().Equal("Feed follow deleted", deleteResponse.Message)
}

func (suite *APITestSuite) TestPosts() {
	// First, create a feed and follow it
	createFeedBody := `{"name":"Test Feed for Posts","url":"http://example.com/rss/feed3.xml"}`
	resp, err := suite.authenticatedClient.Post(suite.server.URL+"/v1/feeds", "application/json", strings.NewReader(createFeedBody))
	suite.Require().NoError(err)
	defer resp.Body.Close()

	// Log the full response for debugging
	respBody, err := io.ReadAll(resp.Body)
	suite.Require().NoError(err)
	suite.T().Logf("Create feed response: %s", string(respBody))

	// Reset the response body for further reading
	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))

	suite.Require().Equal(http.StatusOK, resp.StatusCode, "Failed to create feed")

	var createFeedResponse struct {
		Feed struct {
			Feed struct {
				ID            uuid.UUID  `json:"id"`
				CreatedAt     time.Time  `json:"created_at"`
				UpdatedAt     time.Time  `json:"updated_at"`
				Name          string     `json:"name"`
				URL           string     `json:"url"`
				UserID        uuid.UUID  `json:"userid"`
				LastFetchedAt *time.Time `json:"last_fetched_at"`
			} `json:"feed"`
			FeedFollow struct {
				ID        uuid.UUID `json:"id"`
				CreatedAt time.Time `json:"created_at"`
				UpdatedAt time.Time `json:"updated_at"`
				UserID    uuid.UUID `json:"userid"`
				FeedID    uuid.UUID `json:"feedid"`
			} `json:"feed_follow"`
		} `json:"Feed"`
	}

	err = json.NewDecoder(resp.Body).Decode(&createFeedResponse)
	suite.Require().NoError(err)

	// Log the parsed response
	suite.T().Logf("Parsed create feed response: %+v", createFeedResponse)

	// For the purpose of this test, we'll manually insert some posts into the database
	// In a real scenario, these would be created by the feed scraper
	for i := 0; i < 15; i++ {
		_, err := suite.app.db.CreatePost(context.Background(), database.CreatePostParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Title:     fmt.Sprintf("Test Post %d", i+1),
			Url:       fmt.Sprintf("https://example.com/post%d", i+1),
			Description: sql.NullString{
				String: fmt.Sprintf("Description for Test Post %d", i+1),
				Valid:  true,
			},
			PublishedAt: sql.NullTime{
				Time:  time.Now(),
				Valid: true,
			},
			FeedID: createFeedResponse.Feed.Feed.ID,
		})
		suite.Require().NoError(err)
	}

	// Test getting posts with pagination
	resp, err = suite.authenticatedClient.Get(suite.server.URL + "/v1/posts?page=1&page_size=10")
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Require().Equal(http.StatusOK, resp.StatusCode, "Failed to get posts")

	var getPostsResponse struct {
		Posts []struct {
			ID          uuid.UUID  `json:"id"`
			CreatedAt   time.Time  `json:"created_at"`
			UpdatedAt   time.Time  `json:"updated_at"`
			Title       string     `json:"title"`
			Url         string     `json:"url"`
			Description *string    `json:"description"`
			PublishedAt *time.Time `json:"published_at"`
			FeedID      uuid.UUID  `json:"feedid"`
		} `json:"Posts"`
		Metadata struct {
			CurrentPage  int `json:"current_page"`
			PageSize     int `json:"page_size"`
			FirstPage    int `json:"first_page"`
			LastPage     int `json:"last_page"`
			TotalRecords int `json:"total_records"`
		} `json:"Metadata"`
	}
	err = json.NewDecoder(resp.Body).Decode(&getPostsResponse)
	suite.Require().NoError(err)

	suite.Require().Len(getPostsResponse.Posts, 10, "Should return 10 posts")
	suite.Require().Equal(1, getPostsResponse.Metadata.CurrentPage)
	suite.Require().Equal(10, getPostsResponse.Metadata.PageSize)
	suite.Require().Equal(15, getPostsResponse.Metadata.TotalRecords)

	// Test filtering posts by title
	filterURL := fmt.Sprintf("%s/v1/posts?title=%s", suite.server.URL, url.QueryEscape("Test Post 1"))
	suite.T().Logf("Filtering posts with URL: %s", filterURL)
	resp, err = suite.authenticatedClient.Get(filterURL)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	// Log the response body for debugging
	bodyBytes, err := io.ReadAll(resp.Body)
	suite.Require().NoError(err)
	suite.T().Logf("Filter response body: %s", string(bodyBytes))

	// Reset the response body
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	suite.Require().Equal(http.StatusOK, resp.StatusCode, "Failed to get filtered posts")

	err = json.NewDecoder(resp.Body).Decode(&getPostsResponse)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(getPostsResponse.Posts, "Should return filtered posts")
	for _, post := range getPostsResponse.Posts {
		suite.Require().Contains(post.Title, "Test Post 1", "Filtered posts should contain the search term")
	}

	// Test sorting posts
	resp, err = suite.authenticatedClient.Get(suite.server.URL + "/v1/posts?sort=-published_at")
	suite.Require().NoError(err)
	defer resp.Body.Close()
	suite.Require().Equal(http.StatusOK, resp.StatusCode, "Failed to get sorted posts")

	err = json.NewDecoder(resp.Body).Decode(&getPostsResponse)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(getPostsResponse.Posts, "Should return sorted posts")

	// Verify that posts are sorted by published_at in descending order
	for i := 1; i < len(getPostsResponse.Posts); i++ {
		suite.Require().True(getPostsResponse.Posts[i-1].PublishedAt.After(*getPostsResponse.Posts[i].PublishedAt) ||
			getPostsResponse.Posts[i-1].PublishedAt.Equal(*getPostsResponse.Posts[i].PublishedAt),
			"Posts should be sorted by published_at in descending order")
	}
}
func TestAPISuite(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}

/*
func (suite *APITestSuite) getMailtrapEmails() ([]MailtrapEmail, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	mailtrapURL := os.Getenv("MAILTRAP_MESSAGES")
	if mailtrapURL == "" {
		return nil, fmt.Errorf("MAILTRAP_MESSAGES environment variable is not set")
	}

	req, err := http.NewRequest("GET", mailtrapURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	apiToken := os.Getenv("MAILTRAP_API_TOKEN")
	if apiToken == "" {
		return nil, fmt.Errorf("MAILTRAP_API_TOKEN environment variable is not set")
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Api-Token", apiToken)

	suite.T().Logf("Sending request to Mailtrap API: %s", mailtrapURL)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var emails []MailtrapEmail
	err = json.Unmarshal(body, &emails)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w, body: %s", err, string(body))
	}

	suite.T().Logf("Retrieved %d emails from Mailtrap", len(emails))
	return emails, nil
}

func (suite *APITestSuite) findActivationEmail(toEmail string) (*MailtrapEmail, error) {
	emails, err := suite.getMailtrapEmails()
	if err != nil {
		return nil, fmt.Errorf("failed to get emails: %w", err)
	}

	suite.T().Logf("Searching for activation email for: %s", toEmail)
	for _, email := range emails {
		suite.T().Logf("Checking email: Subject: %s, To: %s", email.Subject, email.ToEmail)
		if email.Subject == "Welcome to Go-Blog-Aggregator!" && email.ToEmail == toEmail {
			suite.T().Logf("Found activation email for: %s", toEmail)
			return &email, nil
		}
	}

	return nil, fmt.Errorf("activation email not found for %s", toEmail)
}
*/
