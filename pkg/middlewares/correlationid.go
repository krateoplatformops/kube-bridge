package middlewares

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

const (
	developmentIdKey = "X-Development-Id"
)

// CorrelationID returns a middleware that add a
// correlation identifier to the HTTP request.
func CorrelationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id := r.Header.Get(developmentIdKey)
		if id == "" {
			// generate new version 4 uuid
			newid := uuid.New()
			id = newid.String()
		}
		// set the id to the request context
		ctx = context.WithValue(ctx, "developmentId", id)
		r = r.WithContext(ctx)
		// fetch the logger from context and update the context
		// with the correlation id value
		log := zerolog.Ctx(ctx)
		log.UpdateContext(func(c zerolog.Context) zerolog.Context {
			return c.Str("developmentId", id)
		})
		// set the response header
		w.Header().Set(developmentIdKey, id)
		next.ServeHTTP(w, r)
	})
}
