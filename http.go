package astikit

import (
	"context"
	"fmt"
	"net/http"
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
