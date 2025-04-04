package astikit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var ErrHTTPSenderUnmarshaledError = errors.New("astikit: unmarshaled error")

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
	timeout    time.Duration
}

// HTTPSenderRetryFunc is a function that decides whether to retry an HTTP request
type HTTPSenderRetryFunc func(resp *http.Response) bool

// HTTPSenderOptions represents HTTPSender options
type HTTPSenderOptions struct {
	Client     HTTPClient
	Logger     StdLogger
	RetryFunc  HTTPSenderRetryFunc
	RetryMax   int
	RetrySleep time.Duration
	Timeout    time.Duration
}

// NewHTTPSender creates a new HTTP sender
func NewHTTPSender(o HTTPSenderOptions) (s *HTTPSender) {
	s = &HTTPSender{
		client:     o.Client,
		l:          AdaptStdLogger(o.Logger),
		retryFunc:  o.RetryFunc,
		retryMax:   o.RetryMax,
		retrySleep: o.RetrySleep,
		timeout:    o.Timeout,
	}
	if s.client == nil {
		s.client = &http.Client{}
	}
	if s.retryFunc == nil {
		s.retryFunc = s.defaultHTTPRetryFunc
	}
	return
}

func (s *HTTPSender) defaultHTTPRetryFunc(resp *http.Response) bool {
	return resp.StatusCode >= http.StatusInternalServerError
}

// Send sends a new *http.Request
func (s *HTTPSender) Send(req *http.Request) (*http.Response, error) {
	return s.SendWithTimeout(req, s.timeout)
}

// SendWithTimeout sends a new *http.Request with a timeout
func (s *HTTPSender) SendWithTimeout(req *http.Request, timeout time.Duration) (*http.Response, error) {
	// Timeout
	if timeout > 0 {
		// Create context
		ctx, cancel := context.WithTimeout(req.Context(), timeout)
		defer cancel()

		// Update request
		req = req.WithContext(ctx)
	}

	// Send
	return s.send(req, timeout)
}

func (s *HTTPSender) send(req *http.Request, timeout time.Duration) (*http.Response, error) {
	// Set name
	name := req.Method + " request"
	if req.URL != nil {
		name += " to " + req.URL.String()
	}

	// Timeout
	if timeout > 0 {
		name += " with timeout " + timeout.String()
	}

	// Loop
	// We start at retryMax + 1 so that it runs at least once even if retryMax == 0
	var resp *http.Response
	var errDo error
	tries := 0
	for retriesLeft := s.retryMax + 1; retriesLeft > 0; retriesLeft-- {
		// Get request name
		nr := name + " (" + strconv.Itoa(s.retryMax-retriesLeft+2) + "/" + strconv.Itoa(s.retryMax+1) + ")"
		tries++

		// Send request
		s.l.Debugf("astikit: sending %s", nr)
		if resp, errDo = s.client.Do(req); errDo != nil {
			// Stop if error is not temporary
			if netError, ok := errDo.(net.Error); !ok || !netError.Timeout() {
				return nil, errDo
			}
		}

		// Request has timed out
		if err := req.Context().Err(); err != nil {
			// Make sure to close response body
			if errDo == nil {
				resp.Body.Close()
			}
			return nil, err
		}

		// Retry
		if errDo != nil || s.retryFunc(resp) {
			if retriesLeft > 1 {
				if errDo == nil {
					resp.Body.Close()
				}
				s.l.Errorf("astikit: sending %s failed, sleeping %s and retrying... (%d retries left): %w", nr, s.retrySleep, retriesLeft-1, errDo)
				time.Sleep(s.retrySleep)
			}
			continue
		}
		return resp, nil
	}

	// Max retries limit reached
	if errDo != nil {
		return nil, errDo
	}
	return resp, nil
}

type HTTPSenderHeaderFunc func(h http.Header)

type HTTPSenderInvalidStatusCodeError struct {
	Err        error
	StatusCode int
}

func (err HTTPSenderInvalidStatusCodeError) Error() string {
	return fmt.Errorf("astikit: validating status code %d failed: %w", err.StatusCode, err.Err).Error()
}

func (err HTTPSenderInvalidStatusCodeError) Is(target error) bool {
	return errors.Is(err.Err, target)
}

type HTTPSenderStatusCodeFunc func(code int) error

// HTTPSendJSONOptions represents SendJSON options
type HTTPSendJSONOptions struct {
	BodyError      any
	BodyIn         any
	BodyOut        any
	Context        context.Context
	HeadersIn      map[string]string
	HeadersOut     HTTPSenderHeaderFunc
	Host           string
	Method         string
	StatusCodeFunc HTTPSenderStatusCodeFunc
	Timeout        time.Duration
	URL            string
}

// SendJSON sends a new JSON HTTP request
func (s *HTTPSender) SendJSON(o HTTPSendJSONOptions) (err error) {
	// Marshal body in
	var bi io.Reader
	if o.BodyIn != nil {
		bb := &bytes.Buffer{}
		if err = json.NewEncoder(bb).Encode(o.BodyIn); err != nil {
			err = fmt.Errorf("astikit: marshaling body in failed: %w", err)
			return
		}
		bi = bb
	}

	// Get context
	ctx := o.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Create request
	var req *http.Request
	if req, err = http.NewRequestWithContext(ctx, o.Method, o.URL, bi); err != nil {
		err = fmt.Errorf("astikit: creating request failed: %w", err)
		return
	}

	// Update host
	if o.Host != "" {
		req.Host = o.Host
	}

	// Add headers
	for k, v := range o.HeadersIn {
		req.Header.Set(k, v)
	}

	// Get timeout
	timeout := s.timeout
	if o.Timeout > 0 {
		timeout = o.Timeout
	}

	// Handle timeout outside the .send() method otherwise context may be cancelled before
	// reading all response body
	if timeout > 0 {
		// Create context
		ctx, cancel := context.WithTimeout(req.Context(), timeout)
		defer cancel()

		// Update request
		req = req.WithContext(ctx)
	}

	// Send request
	var resp *http.Response
	if resp, err = s.send(req, timeout); err != nil {
		err = fmt.Errorf("astikit: sending request failed: %w", err)
		return
	}
	defer resp.Body.Close()

	// Process headers
	if o.HeadersOut != nil {
		o.HeadersOut(resp.Header)
	}

	// Process status code
	fn := HTTPSenderDefaultStatusCodeFunc
	if o.StatusCodeFunc != nil {
		fn = o.StatusCodeFunc
	}
	if err = fn(resp.StatusCode); err != nil {
		// Try unmarshaling error
		if o.BodyError != nil {
			if err2 := json.NewDecoder(resp.Body).Decode(o.BodyError); err2 == nil {
				err = ErrHTTPSenderUnmarshaledError
				return
			}
		}

		// Default error
		err = HTTPSenderInvalidStatusCodeError{
			Err:        err,
			StatusCode: resp.StatusCode,
		}
		return
	}

	// Unmarshal body out
	if o.BodyOut != nil {
		// Read all
		var b []byte
		if b, err = io.ReadAll(resp.Body); err != nil {
			err = fmt.Errorf("astikit: reading all failed: %w", err)
			return
		}

		// Unmarshal
		if err = json.Unmarshal(b, o.BodyOut); err != nil {
			err = fmt.Errorf("astikit: unmarshaling failed: %w (json: %s)", err, b)
			return
		}
	}
	return
}

func HTTPSenderDefaultStatusCodeFunc(code int) error {
	if code < 200 || code > 299 {
		return errors.New("astikit: status code should be between 200 and 299")
	}
	return nil
}

// HTTPResponseFunc is a func that can process an $http.Response
type HTTPResponseFunc func(resp *http.Response) error

func defaultHTTPResponseFunc(resp *http.Response) (err error) {
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		err = fmt.Errorf("astikit: invalid status code %d", resp.StatusCode)
		return
	}
	return
}

// HTTPDownloader represents an object capable of downloading several HTTP srcs simultaneously
// and doing stuff to the results
type HTTPDownloader struct {
	bp           *BufferPool
	l            *GoroutineLimiter
	responseFunc HTTPResponseFunc
	s            *HTTPSender
}

// HTTPDownloaderOptions represents HTTPDownloader options
type HTTPDownloaderOptions struct {
	Limiter      GoroutineLimiterOptions
	ResponseFunc HTTPResponseFunc
	Sender       HTTPSenderOptions
}

// NewHTTPDownloader creates a new HTTPDownloader
func NewHTTPDownloader(o HTTPDownloaderOptions) (d *HTTPDownloader) {
	d = &HTTPDownloader{
		bp:           NewBufferPool(),
		l:            NewGoroutineLimiter(o.Limiter),
		responseFunc: o.ResponseFunc,
		s:            NewHTTPSender(o.Sender),
	}
	if d.responseFunc == nil {
		d.responseFunc = defaultHTTPResponseFunc
	}
	return
}

// Close closes the downloader properly
func (d *HTTPDownloader) Close() error {
	return d.l.Close()
}

type HTTPDownloaderSrc struct {
	Body   io.Reader
	Header http.Header
	Method string
	URL    string
}

// It is the responsibility of the caller to call i.Close()
type httpDownloaderFunc func(ctx context.Context, idx int, i *BufferPoolItem) error

func (d *HTTPDownloader) do(ctx context.Context, fn httpDownloaderFunc, idx int, src HTTPDownloaderSrc) (err error) {
	// Defaults
	if src.Method == "" {
		src.Method = http.MethodGet
	}

	// Create request
	var r *http.Request
	if r, err = http.NewRequestWithContext(ctx, src.Method, src.URL, src.Body); err != nil {
		err = fmt.Errorf("astikit: creating request to %s failed: %w", src.URL, err)
		return
	}

	// Copy header
	for k := range src.Header {
		r.Header.Set(k, src.Header.Get(k))
	}

	// Send request
	var resp *http.Response
	if resp, err = d.s.Send(r); err != nil {
		err = fmt.Errorf("astikit: sending request to %s failed: %w", src.URL, err)
		return
	}
	defer resp.Body.Close()

	// Create buffer pool item
	buf := d.bp.New()

	// Process response
	if err = d.responseFunc(resp); err != nil {
		err = fmt.Errorf("astikit: response for request to %s is invalid: %w", src.URL, err)
		return
	}

	// Copy body
	if _, err = Copy(ctx, buf, resp.Body); err != nil {
		err = fmt.Errorf("astikit: copying body of %s failed: %w", src.URL, err)
		return
	}

	// Custom
	if err = fn(ctx, idx, buf); err != nil {
		err = fmt.Errorf("astikit: custom callback on %s failed: %w", src.URL, err)
		return
	}
	return
}

func (d *HTTPDownloader) download(ctx context.Context, srcs []HTTPDownloaderSrc, fn httpDownloaderFunc) (err error) {
	// Nothing to download
	if len(srcs) == 0 {
		return nil
	}

	// Loop through srcs
	wg := &sync.WaitGroup{}
	wg.Add(len(srcs))
	for idx, src := range srcs {
		func(idx int, src HTTPDownloaderSrc) {
			// Update error with ctx
			if ctx.Err() != nil {
				err = ctx.Err()
			}

			// Do nothing if error
			if err != nil {
				wg.Done()
				return
			}

			// Do
			//nolint:errcheck
			d.l.Do(func() {
				// Task is done
				defer wg.Done()

				// Do
				if errD := d.do(ctx, fn, idx, src); errD != nil && err == nil {
					err = errD
					return
				}
			})
		}(idx, src)
	}

	// Wait
	wg.Wait()
	return
}

// DownloadInDirectory downloads in parallel a set of srcs and saves them in a dst directory
func (d *HTTPDownloader) DownloadInDirectory(ctx context.Context, dst string, srcs ...HTTPDownloaderSrc) error {
	return d.download(ctx, srcs, func(ctx context.Context, idx int, buf *BufferPoolItem) (err error) {
		// Make sure to close buffer
		defer buf.Close()

		// Make sure destination directory exists
		if err = os.MkdirAll(dst, DefaultDirMode); err != nil {
			err = fmt.Errorf("astikit: mkdirall %s failed: %w", dst, err)
			return
		}

		// Create destination file
		var f *os.File
		dst := filepath.Join(dst, filepath.Base(srcs[idx].URL))
		if f, err = os.Create(dst); err != nil {
			err = fmt.Errorf("astikit: creating %s failed: %w", dst, err)
			return
		}
		defer f.Close()

		// Copy buffer
		if _, err = Copy(ctx, f, buf); err != nil {
			err = fmt.Errorf("astikit: copying content to %s failed: %w", dst, err)
			return
		}
		return
	})
}

// DownloadInWriter downloads in parallel a set of srcs and concatenates them in a writer while
// maintaining the initial order
func (d *HTTPDownloader) DownloadInWriter(ctx context.Context, dst io.Writer, srcs ...HTTPDownloaderSrc) error {
	// Init
	type chunk struct {
		buf *BufferPoolItem
		idx int
	}
	var cs []chunk
	var m sync.Mutex // Locks cs
	var requiredIdx int

	// Make sure to close all buffers
	defer func() {
		for _, c := range cs {
			c.buf.Close()
		}
	}()

	// Download
	return d.download(ctx, srcs, func(ctx context.Context, idx int, buf *BufferPoolItem) (err error) {
		// Lock
		m.Lock()
		defer m.Unlock()

		// Check where to insert chunk
		var idxInsert = -1
		for idxChunk := 0; idxChunk < len(cs); idxChunk++ {
			if idx < cs[idxChunk].idx {
				idxInsert = idxChunk
				break
			}
		}

		// Create chunk
		c := chunk{
			buf: buf,
			idx: idx,
		}

		// Add chunk
		if idxInsert > -1 {
			cs = append(cs[:idxInsert], append([]chunk{c}, cs[idxInsert:]...)...)
		} else {
			cs = append(cs, c)
		}

		// Loop through chunks
		for idxChunk := 0; idxChunk < len(cs); idxChunk++ {
			// Get chunk
			c := cs[idxChunk]

			// The chunk should be copied
			if c.idx == requiredIdx {
				// Copy chunk content
				// Do not check error right away since we still want to close the buffer
				// and remove the chunk
				_, err = Copy(ctx, dst, c.buf)

				// Close buffer
				c.buf.Close()

				// Remove chunk
				requiredIdx++
				cs = append(cs[:idxChunk], cs[idxChunk+1:]...)
				idxChunk--

				// Check error
				if err != nil {
					err = fmt.Errorf("astikit: copying chunk #%d to dst failed: %w", c.idx, err)
					return
				}
			}
		}
		return
	})
}

// DownloadInFile downloads in parallel a set of srcs and concatenates them in a dst file while
// maintaining the initial order
func (d *HTTPDownloader) DownloadInFile(ctx context.Context, dst string, srcs ...HTTPDownloaderSrc) (err error) {
	// Make sure destination directory exists
	if err = os.MkdirAll(filepath.Dir(dst), DefaultDirMode); err != nil {
		err = fmt.Errorf("astikit: mkdirall %s failed: %w", filepath.Dir(dst), err)
		return
	}

	// Create destination file
	var f *os.File
	if f, err = os.Create(dst); err != nil {
		err = fmt.Errorf("astikit: creating %s failed: %w", dst, err)
		return
	}
	defer f.Close()

	// Download in writer
	return d.DownloadInWriter(ctx, f, srcs...)
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
		if m == nil {
			continue
		}
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
	if u, p, ok := r.BasicAuth(); !ok || u != username || p != password {
		rw.Header().Set("WWW-Authenticate", "Basic Realm=Please enter your credentials")
		rw.WriteHeader(http.StatusUnauthorized)
		return true
	}
	return false
}

// HTTPMiddlewareBasicAuth adds basic HTTP auth to an HTTP handler
func HTTPMiddlewareBasicAuth(username, password string) HTTPMiddleware {
	if username == "" && password == "" {
		return nil
	}
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			// Handle basic auth
			if handleHTTPBasicAuth(username, password, rw, r) {
				return
			}

			// Next handler
			h.ServeHTTP(rw, r)
		})
	}
}

func setHTTPContentType(contentType string, rw http.ResponseWriter) {
	rw.Header().Set("Content-Type", contentType)
}

// HTTPMiddlewareContentType adds a content type to an HTTP handler
func HTTPMiddlewareContentType(contentType string) HTTPMiddleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			// Set content type
			setHTTPContentType(contentType, rw)

			// Next handler
			h.ServeHTTP(rw, r)
		})
	}
}

func setHTTPHeaders(vs map[string]string, rw http.ResponseWriter) {
	for k, v := range vs {
		rw.Header().Set(k, v)
	}
}

// HTTPMiddlewareHeaders adds headers to an HTTP handler
func HTTPMiddlewareHeaders(vs map[string]string) HTTPMiddleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			// Set headers
			setHTTPHeaders(vs, rw)

			// Next handler
			h.ServeHTTP(rw, r)
		})
	}
}

// HTTPMiddlewareCORSHeaders adds CORS headers to an HTTP handler
func HTTPMiddlewareCORSHeaders() HTTPMiddleware {
	return HTTPMiddlewareHeaders(map[string]string{
		"Access-Control-Allow-Headers": "*",
		"Access-Control-Allow-Methods": "*",
		"Access-Control-Allow-Origin":  "*",
	})
}
