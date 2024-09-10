package main

import (
	"expvar"
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

	router.HandlerFunc(http.MethodPost, "/v1/feeds", app.requirePermission("feeds:write", app.HandlerFeedsCreate))
	router.HandlerFunc(http.MethodGet, "/v1/feeds", app.requirePermission("feeds:read", app.HandlerFeedsGet))

	router.HandlerFunc(http.MethodPost, "/v1/feed_follows", app.requirePermission("feed_follows:write", app.HandlerFeedFollowsCreate))
	router.HandlerFunc(http.MethodDelete, "/v1/feed_follows/:feedfollowID", app.requirePermission("feed_follows:write", app.HandlerFeedFollowsDelete))
	router.HandlerFunc(http.MethodGet, "/v1/feed_follows", app.requirePermission("feed_follows:read", app.HandlerFeedFollowsGet))

	router.HandlerFunc(http.MethodGet, "/v1/posts", app.requirePermission("posts:read", app.HandlerPostsGet))

	router.Handler(http.MethodGet, "/debug/vars", expvar.Handler())

	return app.metrics(app.recoverPanic(app.enableCORS(app.rateLimit(app.authenticate(router)))))
}
