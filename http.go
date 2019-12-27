package astikit

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
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

// HTTPClient represents an HTTP client
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// HTTPSender represents an object capable of sending http requests
type HTTPSender struct {
	client     HTTPClient
	l          SeverityLogger
	retryFunc  HTTPSenderRetryFunc
	retryMax   int
	retrySleep time.Duration
}

// HTTPSenderRetryFunc is a function that decides whether to retry an HTTP request
type HTTPSenderRetryFunc func(name string, resp *http.Response) bool

// HTTPSenderOptions represents HTTPSender options
type HTTPSenderOptions struct {
	Client     HTTPClient
	Logger     StdLogger
	RetryFunc  HTTPSenderRetryFunc
	RetryMax   int
	RetrySleep time.Duration
}

// NewHTTPSender creates a new HTTP sender
func NewHTTPSender(o HTTPSenderOptions) (s *HTTPSender) {
	s = &HTTPSender{
		client:     o.Client,
		l:          AdaptStdLogger(o.Logger),
		retryFunc:  o.RetryFunc,
		retryMax:   o.RetryMax,
		retrySleep: o.RetrySleep,
	}
	if s.client == nil {
		s.client = &http.Client{}
	}
	if s.retryFunc == nil {
		s.retryFunc = s.defaultHTTPRetryFunc
	}
	return
}

func (s *HTTPSender) defaultHTTPRetryFunc(name string, resp *http.Response) bool {
	if resp.StatusCode >= http.StatusInternalServerError {
		s.l.Debugf("astikit: invalid status code %d when sending %s", resp.StatusCode, name)
		return true
	}
	return false
}

// Send sends a new *http.Request
func (s *HTTPSender) Send(req *http.Request) (resp *http.Response, err error) {
	return s.execWithRetry(fmt.Sprintf("%s request to %s", req.Method, req.URL), func() (*http.Response, error) { return s.client.Do(req) })
}

// name is used for logging purposes only
func (s *HTTPSender) execWithRetry(name string, fn func() (*http.Response, error)) (resp *http.Response, err error) {
	// Loop
	// We start at retryMax + 1 so that it runs at least once even if retryMax == 0
	tries := 0
	for retriesLeft := s.retryMax + 1; retriesLeft > 0; retriesLeft-- {
		// Get request name
		nr := fmt.Sprintf("%s (%d/%d)", name, s.retryMax-retriesLeft+2, s.retryMax+1)
		tries++

		// Send request
		var retry bool
		s.l.Debugf("astikit: sending %s", nr)
		if resp, err = fn(); err != nil {
			// If error is temporary, retry
			if netError, ok := err.(net.Error); ok && netError.Temporary() {
				s.l.Debugf("astikit: temporary error when sending %s", nr)
				retry = true
			} else {
				err = fmt.Errorf("astikit: sending %s failed: %w", nr, err)
				return
			}
		}

		// Retry
		if retry || s.retryFunc(nr, resp) {
			if retriesLeft > 1 {
				s.l.Debugf("astikit: sleeping %s and retrying... (%d retries left)", s.retrySleep, retriesLeft-1)
				time.Sleep(s.retrySleep)
			}
			continue
		}

		// Return if conditions for retrying were not met
		return
	}

	// Max retries limit reached
	err = fmt.Errorf("astikit: sending %s failed after %d tries", name, tries)
	return
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
