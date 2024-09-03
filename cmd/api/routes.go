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

	router.HandlerFunc(http.MethodPost, "/v1/feeds", app.MiddlewareAuth(app.HandlerFeedsCreate))
	router.HandlerFunc(http.MethodGet, "/v1/feeds", app.HandlerFeedsGet)

	router.HandlerFunc(http.MethodPost, "/v1/feed_follows", app.MiddlewareAuth(app.HandlerFeedFollowsCreate))
	router.HandlerFunc(http.MethodDelete, "/v1/feed_follows/:feedfollowID", app.MiddlewareAuth(app.HandlerFeedFollowsDelete))
	router.HandlerFunc(http.MethodGet, "/v1/feed_follows", app.MiddlewareAuth(app.HandlerFeedFollowsGet))

	router.HandlerFunc(http.MethodGet, "/v1/posts", app.MiddlewareAuth(app.HandlerPostsGet))

	return router
}
