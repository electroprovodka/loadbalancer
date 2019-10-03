package main

import (
	"context"
	"net/http"

	"github.com/rs/xid"
	log "github.com/sirupsen/logrus"
)

// Middleware is a function that accepts allows to add additional behavior to the request processing cycle
// Middleware should return new http.Handler, that calls the http.Handler that was passed to it as `next` parameter
type Middleware func(next http.Handler) http.Handler

type key int

var requestIDKey key = 0

func getRequestID() string {
	return xid.New().String()
}

// Apply list of middlewares for router
// Note: we apply middlewares the way that the first middleware in list will be the firs middleware to receive the request
func applyMiddlewares(router http.Handler, middlewares []Middleware) http.Handler {
	n := len(middlewares)
	// Applying middlewares in the reverse order, so the first middleware in the list will be the outmost
	for i := n - 1; i >= 0; i-- {
		router = middlewares[i](router)
	}
	return router
}

func tracing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-Id")
		if requestID == "" {
			requestID = getRequestID()
		}
		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		r.Header.Set("X-Request-Id", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			requestID, ok := r.Context().Value(requestIDKey).(string)
			if !ok {
				// TODO: generate new ID?
				requestID = "unknown"
			}
			// TODO: log response code
			// TODO: log the target redirect
			log.Printf("[ID:%s] %s %s, %s, %s", requestID, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
		}()
		next.ServeHTTP(w, r)
	})
}
