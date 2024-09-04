package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/v1/healthz", app.healthcheckHandler)

	router.HandlerFunc(http.MethodPost, "/v1/users", app.HandlerUsersCreate)
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.HandlerUserActivate)

	router.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.HandlerAuthenticationTokenCreate)

	router.HandlerFunc(http.MethodPost, "/v1/feeds", app.requireActivatedUser(app.HandlerFeedsCreate))
	router.HandlerFunc(http.MethodGet, "/v1/feeds", app.requireActivatedUser(app.HandlerFeedsGet))

	router.HandlerFunc(http.MethodPost, "/v1/feed_follows", app.requireActivatedUser(app.HandlerFeedFollowsCreate))
	router.HandlerFunc(http.MethodDelete, "/v1/feed_follows/:feedfollowID", app.requireActivatedUser(app.HandlerFeedFollowsDelete))
	router.HandlerFunc(http.MethodGet, "/v1/feed_follows", app.requireActivatedUser(app.HandlerFeedFollowsGet))

	router.HandlerFunc(http.MethodGet, "/v1/posts", app.requireActivatedUser(app.HandlerPostsGet))

	return app.recoverPanic(app.authenticate(router))
}
