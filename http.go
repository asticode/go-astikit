package astikit

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// ServeHTTPOptions represents serve options
type ServeHTTPOptions struct {
	Addr    string
	Handler http.Handler
}

// ServeHTTP spawns an HTTP server
func ServeHTTP(w *Worker, o ServeHTTPOptions) {
	// Create server
	s := &http.Server{Addr: o.Addr, Handler: o.Handler}

	// Execute in a task
	w.NewTask().Do(func() {
		// Log
		w.Logger().Infof("astikit: serving on %s", o.Addr)

		// Serve
		var done = make(chan error)
		go func() {
			if err := s.ListenAndServe(); err != nil {
				done <- err
			}
		}()

		// Wait for context or done to be done
		select {
		case <-w.ctx.Done():
			if w.ctx.Err() != context.Canceled {
				w.Logger().Error(fmt.Errorf("astikit: context error: %w", w.ctx.Err()))
			}
		case err := <-done:
			if err != nil {
				w.Logger().Error(fmt.Errorf("astikit: serving failed: %w", err))
			}
		}

		// Shutdown
		w.Logger().Infof("astikit: shutting down server on %s", o.Addr)
		if err := s.Shutdown(context.Background()); err != nil {
			w.Logger().Error(fmt.Errorf("astikit: shutting down server on %s failed: %w", o.Addr, err))
		}
	})
}

// HTTPMiddleware represents an HTTP middleware
type HTTPMiddleware func(http.Handler) http.Handler

// ChainHTTPMiddlewares chains HTTP middlewares
func ChainHTTPMiddlewares(h http.Handler, ms ...HTTPMiddleware) http.Handler {
	return ChainHTTPMiddlewaresWithPrefix(h, []string{}, ms...)
}

// ChainHTTPMiddlewaresWithPrefix chains HTTP middlewares if one of prefixes is present
func ChainHTTPMiddlewaresWithPrefix(h http.Handler, prefixes []string, ms ...HTTPMiddleware) http.Handler {
	for _, m := range ms {
		if len(prefixes) == 0 {
			h = m(h)
		} else {
			t := h
			h = http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				for _, prefix := range prefixes {
					if strings.HasPrefix(r.URL.EscapedPath(), prefix) {
						m(t).ServeHTTP(rw, r)
						return
					}
				}
				t.ServeHTTP(rw, r)
			})
		}
	}
	return h
}

func handleHTTPBasicAuth(username, password string, rw http.ResponseWriter, r *http.Request) bool {
	if len(username) > 0 && len(password) > 0 {
		if u, p, ok := r.BasicAuth(); !ok || u != username || p != password {
			rw.Header().Set("WWW-Authenticate", "Basic Realm=Please enter your credentials")
			rw.WriteHeader(http.StatusUnauthorized)
			return true
		}
	}
	return false
}

// HTTPMiddlewareBasicAuth adds basic HTTP auth to an HTTP handler
func HTTPMiddlewareBasicAuth(username, password string) HTTPMiddleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			// Basic auth
			if handleHTTPBasicAuth(username, password, rw, r) {
				return
			}

			// Next handler
			h.ServeHTTP(rw, r)
		})
	}
}

func handleHTTPContentType(contentType string, rw http.ResponseWriter) {
	rw.Header().Set("Content-Type", contentType)
}

// HTTPMiddlewareContentType adds a content type to an HTTP handler
func HTTPMiddlewareContentType(contentType string) HTTPMiddleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			// Content type
			handleHTTPContentType(contentType, rw)

			// Next handler
			h.ServeHTTP(rw, r)
		})
	}
}
