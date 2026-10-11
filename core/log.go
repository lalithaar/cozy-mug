package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"
)

// Options configures the default middleware.
type Options struct {
	// Dev enables verbose error pages and richer output.
	Dev    bool
	Logger *slog.Logger // nil uses slog.Default()
}

func (o Options) logger() *slog.Logger {
	if o.Logger != nil {
		return o.Logger
	}
	return slog.Default()
}

type reqIDKey struct{}

// RequestID returns the id assigned by the Logger middleware, or "".
func (c *Context) RequestID() string {
	id, _ := c.R.Context().Value(reqIDKey{}).(string)
	return id
}

// Logger logs one line per request (method, path, status, duration, id) and
// assigns a request id (also sent as X-Request-Id). Place it outermost so it
// sees the final status, after the Errors middleware has rendered error pages.
func Logger(o Options) Middleware {
	log := o.logger()
	return func(next Handler) Handler {
		return func(c *Context) Response {
			id := newRequestID()
			if id != "" {
				c.R = c.R.WithContext(context.WithValue(c.R.Context(), reqIDKey{}, id))
				c.W.Header().Set("X-Request-Id", id)
			}
			start := time.Now()
			resp := next(c)
			status := http.StatusOK
			if sc, ok := resp.(StatusCoder); ok {
				status = sc.StatusCode()
			}
			log.Info("request",
				"method", c.Method(),
				"path", c.Path(),
				"status", status,
				"dur", time.Since(start).Round(time.Microsecond),
				"id", id,
			)
			return resp
		}
	}
}

func newRequestID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return ""
	}
	return hex.EncodeToString(b[:])
}
