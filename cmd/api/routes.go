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

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)

	router.HandlerFunc(http.MethodGet, "/v1/images", app.serveImages)
	router.HandlerFunc(http.MethodGet, "/v1/pdfs", app.servePdfs)
	router.HandlerFunc(http.MethodGet, "/v1/epubs", app.serveEpubs)
	router.HandlerFunc(http.MethodGet, "/v1/torrs", app.serveTorrents)

	router.HandlerFunc(http.MethodGet, "/v1/books", app.listBookHandler)
	router.HandlerFunc(http.MethodPost, "/v1/books", app.requirePermission("books:write", app.createBookHandler))
	router.HandlerFunc(http.MethodGet, "/v1/books/:slug", app.showBookHandler)
	router.HandlerFunc(http.MethodPatch, "/v1/books/:id", app.requirePermission("books:write", app.updateBookHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/books/:id", app.requirePermission("books:write", app.deleteMovieHandler))

	router.HandlerFunc(http.MethodPost, "/v1/authors", app.requirePermission("books:write", app.createAuthorHandler))
	router.HandlerFunc(http.MethodGet, "/v1/authors", app.listAuthorsHandler)
	router.HandlerFunc(http.MethodGet, "/v1/authors/:id", app.showAuthorHandler)

	router.HandlerFunc(http.MethodPost, "/v1/publishers", app.requirePermission("books:write", app.createPublisherHandler))
	router.HandlerFunc(http.MethodGet, "/v1/publishers", app.listPublishersHandler)
	router.HandlerFunc(http.MethodGet, "/v1/publishers/:id", app.showPublisherHandler)

	router.HandlerFunc(http.MethodGet, "/v1/tags", app.listTagsHandler)

	router.HandlerFunc(http.MethodPost, "/v1/users", app.registerUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.activateUserHandler)
	router.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.createAuthenticationTokenHandler)

	router.Handler(http.MethodGet, "/v1/metrics", expvar.Handler())

	return app.metrics(app.recoverPanic(app.enableCORS(app.rateLimit(app.authenticate(router)))))
}
