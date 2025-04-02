package astikit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestServeHTTP(t *testing.T) {
	w := NewWorker(WorkerOptions{})
	ln, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	ln.Close()
	var i int
	ServeHTTP(w, ServeHTTPOptions{
		Addr: ln.Addr().String(),
		Handler: http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			i++
			w.Stop()
		}),
	})
	s := time.Now()
	for {
		if time.Since(s) > time.Second {
			t.Fatal("timed out")
		}

		_, err := http.DefaultClient.Get("http://" + ln.Addr().String())
		if err != nil {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		break
	}
	w.Wait()
	if e := 1; i != e {
		t.Fatalf("expected %+v, got %+v", e, i)
	}
}

type mockedHTTPClient func(req *http.Request) (*http.Response, error)

func (c mockedHTTPClient) Do(req *http.Request) (*http.Response, error) { return c(req) }

type mockedNetError struct{ timeout bool }

func (err mockedNetError) Error() string   { return "" }
func (err mockedNetError) Timeout() bool   { return err.timeout }
func (err mockedNetError) Temporary() bool { return false }

type mockedHTTPBody struct {
	closed bool
}

func (b *mockedHTTPBody) Read([]byte) (int, error) {
	return 0, nil
}

func (b *mockedHTTPBody) Close() error {
	b.closed = true
	return nil
}

func TestHTTPSender(t *testing.T) {
	// All errors
	var c int
	var bs []*mockedHTTPBody
	s1 := NewHTTPSender(HTTPSenderOptions{
		Client: mockedHTTPClient(func(req *http.Request) (resp *http.Response, err error) {
			c++
			b := &mockedHTTPBody{}
			bs = append(bs, b)
			resp = &http.Response{
				Body:       b,
				StatusCode: http.StatusInternalServerError,
			}
			return
		}),
		RetryMax: 3,
	})
	if _, err := s1.Send(&http.Request{}); err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	if e := 4; c != e {
		t.Fatalf("expected %v, got %v", e, c)
	}
	if e, g := 4, len(bs); e != g {
		t.Fatalf("expected %v, got %v", e, g)
	}
	for i := 0; i < len(bs)-1; i++ {
		if !bs[i].closed {
			t.Fatalf("body #%d is not closed", i+1)
		}
	}

	// Successful after retries
	bs = []*mockedHTTPBody{}
	c = 0
	s2 := NewHTTPSender(HTTPSenderOptions{
		Client: mockedHTTPClient(func(req *http.Request) (resp *http.Response, err error) {
			c++
			switch c {
			case 1:
				b := &mockedHTTPBody{}
				bs = append(bs, b)
				resp = &http.Response{
					Body:       b,
					StatusCode: http.StatusInternalServerError,
				}
			case 2:
				err = mockedNetError{timeout: true}
			default:
				// No retrying
				b := &mockedHTTPBody{}
				bs = append(bs, b)
				resp = &http.Response{
					Body:       b,
					StatusCode: http.StatusBadRequest,
				}
			}
			return
		}),
		RetryMax: 3,
	})
	if _, err := s2.Send(&http.Request{}); err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	if e := 3; c != e {
		t.Fatalf("expected %v, got %v", e, c)
	}
	if e, g := 2, len(bs); e != g {
		t.Fatalf("expected %v, got %v", e, g)
	}
	for i := 0; i < len(bs)-1; i++ {
		if !bs[i].closed {
			t.Fatalf("body #%d is not closed", i+1)
		}
	}

	// JSON
	var (
		ebe = "error"
		ebi = "body-in"
		ebo = "body-out"
		ehi = map[string]string{
			"K1": "v1",
			"K2": "v2",
		}
		eho = http.Header{"k1": []string{"v1"}}
		eu  = "https://domain.com/url"
	)
	var gu, gbi string
	ghi := make(map[string]string)
	s3 := NewHTTPSender(HTTPSenderOptions{
		Client: mockedHTTPClient(func(req *http.Request) (resp *http.Response, err error) {
			switch req.Method {
			case http.MethodHead:
				for k, v := range req.Header {
					ghi[k] = strings.Join(v, ",")
				}
				gu = req.URL.String()
				resp = &http.Response{
					Body:       io.NopCloser(&bytes.Buffer{}),
					Header:     eho,
					StatusCode: http.StatusBadRequest,
				}
			case http.MethodPost:
				json.NewDecoder(req.Body).Decode(&gbi) //nolint:errcheck
				resp = &http.Response{Body: io.NopCloser(bytes.NewBuffer([]byte("\"" + ebe + "\""))), StatusCode: http.StatusBadRequest}
			case http.MethodGet:
				resp = &http.Response{Body: io.NopCloser(bytes.NewBuffer([]byte("\"" + ebo + "\""))), StatusCode: http.StatusOK}
			}
			return
		}),
	})
	var gho http.Header
	errTest := errors.New("test")
	var isce HTTPSenderInvalidStatusCodeError
	if err := s3.SendJSON(HTTPSendJSONOptions{
		HeadersIn:  ehi,
		HeadersOut: func(h http.Header) { gho = h },
		Method:     http.MethodHead,
		StatusCodeFunc: func(code int) error {
			if code == http.StatusBadRequest {
				return errTest
			}
			return nil
		},
		URL: eu,
	}); err == nil {
		t.Fatal("expected error, got nil")
	} else if !errors.Is(err, errTest) {
		t.Fatal("expected true, got false")
	} else if !errors.As(err, &isce) {
		t.Fatal("expected true, got false")
	}
	if e, g := (HTTPSenderInvalidStatusCodeError{
		Err:        errTest,
		StatusCode: http.StatusBadRequest,
	}), isce; !reflect.DeepEqual(e, g) {
		t.Fatalf("expected %+v, got %+v", e, g)
	}
	if !reflect.DeepEqual(ehi, ghi) {
		t.Fatalf("expected %+v, got %+v", ehi, ghi)
	}
	if !reflect.DeepEqual(eho, gho) {
		t.Fatalf("expected %+v, got %+v", eho, gho)
	}
	if gu != eu {
		t.Fatalf("expected %s, got %s", eu, gu)
	}
	var gbe string
	if err := s3.SendJSON(HTTPSendJSONOptions{
		BodyError: &gbe,
		BodyIn:    ebi,
		Method:    http.MethodPost,
	}); !errors.Is(err, ErrHTTPSenderUnmarshaledError) {
		t.Fatalf("expected ErrHTTPSenderUnmarshaledError, got %s", err)
	}
	if gbe != ebe {
		t.Fatalf("expected %s, got %s", ebe, gbe)
	}
	if gbi != ebi {
		t.Fatalf("expected %s, got %s", ebi, gbi)
	}
	var gbo string
	if err := s3.SendJSON(HTTPSendJSONOptions{
		BodyOut: &gbo,
		Method:  http.MethodGet,
	}); err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	if gbo != ebo {
		t.Fatalf("expected %s, got %s", ebo, gbo)
	}

	// Timeout
	bs = []*mockedHTTPBody{}
	timeoutMockedHTTPClient := mockedHTTPClient(func(req *http.Request) (resp *http.Response, err error) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		<-ctx.Done()
		b := &mockedHTTPBody{}
		bs = append(bs, b)
		resp = &http.Response{Body: b}
		return
	})
	s4 := NewHTTPSender(HTTPSenderOptions{Client: timeoutMockedHTTPClient})
	if _, err := s4.SendWithTimeout(&http.Request{}, time.Millisecond); err == nil {
		t.Fatal("expected error, got nil")
	}
	if err := s4.SendJSON(HTTPSendJSONOptions{Timeout: time.Millisecond}); err == nil {
		t.Fatal("expected error, got nil")
	}
	if e, g := 2, len(bs); e != g {
		t.Fatalf("expected %v, got %v", e, g)
	}
	for i, b := range bs {
		if !b.closed {
			t.Fatalf("body #%d is not closed", i+1)
		}
	}
	// Make sure reading response body doesn't fail if timeout is not reached
	if err := s3.SendJSON(HTTPSendJSONOptions{
		BodyOut: &gbo,
		Method:  http.MethodGet,
		Timeout: time.Hour,
	}); err != nil {
		t.Fatalf("expected no error, got %s", err)
	}

	// Context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ctxCheckerMockedHTTPClient := mockedHTTPClient(func(req *http.Request) (resp *http.Response, err error) {
		return &http.Response{}, req.Context().Err()
	})
	s5 := NewHTTPSender(HTTPSenderOptions{Client: ctxCheckerMockedHTTPClient})
	if err := s5.SendJSON(HTTPSendJSONOptions{Context: ctx}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancelled error, got %s", err)
	}
}

func TestHTTPDownloader(t *testing.T) {
	// Get temp dir
	dir := t.TempDir()

	// Create downloader
	d := NewHTTPDownloader(HTTPDownloaderOptions{
		Limiter: GoroutineLimiterOptions{Max: 2},
		Sender: HTTPSenderOptions{
			Client: mockedHTTPClient(func(req *http.Request) (resp *http.Response, err error) {
				// In case of DownloadInWriter we want to check if the order is kept event
				// if downloaded order is messed up
				if req.URL.EscapedPath() == "/path/to/2" {
					time.Sleep(time.Millisecond)
				}
				resp = &http.Response{
					Body:       io.NopCloser(bytes.NewBufferString(req.URL.EscapedPath())),
					StatusCode: http.StatusOK,
				}
				return
			}),
		},
	})
	defer d.Close()

	// Download in directory
	err := d.DownloadInDirectory(context.Background(), dir,
		HTTPDownloaderSrc{URL: "/path/to/1"},
		HTTPDownloaderSrc{URL: "/path/to/2"},
		HTTPDownloaderSrc{URL: "/path/to/3"},
	)
	if err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	checkDir(t, dir, map[string]string{
		"/1": "/path/to/1",
		"/2": "/path/to/2",
		"/3": "/path/to/3",
	})

	// Download in writer
	w := &bytes.Buffer{}
	err = d.DownloadInWriter(context.Background(), w,
		HTTPDownloaderSrc{URL: "/path/to/1"},
		HTTPDownloaderSrc{URL: "/path/to/2"},
		HTTPDownloaderSrc{URL: "/path/to/3"},
	)
	if err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	if e, g := "/path/to/1/path/to/2/path/to/3", w.String(); e != g {
		t.Fatalf("expected %s, got %s", e, g)
	}

	// Download in file
	p := filepath.Join(dir, "f")
	err = d.DownloadInFile(context.Background(), p,
		HTTPDownloaderSrc{URL: "/path/to/1"},
		HTTPDownloaderSrc{URL: "/path/to/2"},
		HTTPDownloaderSrc{URL: "/path/to/3"},
	)
	if err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	checkFile(t, p, "/path/to/1/path/to/2/path/to/3")
}
