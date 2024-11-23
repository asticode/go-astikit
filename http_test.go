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

func TestHTTPSender(t *testing.T) {
	// All errors
	var c int
	s := NewHTTPSender(HTTPSenderOptions{
		Client: mockedHTTPClient(func(req *http.Request) (resp *http.Response, err error) {
			c++
			resp = &http.Response{StatusCode: http.StatusInternalServerError}
			return
		}),
		RetryMax: 3,
	})
	if _, err := s.Send(&http.Request{}); err == nil {
		t.Fatal("expected error, got nil")
	}
	if e := 4; c != e {
		t.Fatalf("expected %v, got %v", e, c)
	}

	// Successful after retries
	c = 0
	s = NewHTTPSender(HTTPSenderOptions{
		Client: mockedHTTPClient(func(req *http.Request) (resp *http.Response, err error) {
			c++
			switch c {
			case 1:
				resp = &http.Response{StatusCode: http.StatusInternalServerError}
			case 2:
				err = mockedNetError{timeout: true}
			default:
				// No retrying
				resp = &http.Response{StatusCode: http.StatusBadRequest}
			}
			return
		}),
		RetryMax: 3,
	})
	if _, err := s.Send(&http.Request{}); err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	if e := 3; c != e {
		t.Fatalf("expected %v, got %v", e, c)
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
	s = NewHTTPSender(HTTPSenderOptions{
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
	if err := s.SendJSON(HTTPSendJSONOptions{
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
	if err := s.SendJSON(HTTPSendJSONOptions{
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
	if err := s.SendJSON(HTTPSendJSONOptions{
		BodyOut: &gbo,
		Method:  http.MethodGet,
	}); err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	if gbo != ebo {
		t.Fatalf("expected %s, go %s", ebo, gbo)
	}

	// Timeout
	timeoutMockedHTTPClient := mockedHTTPClient(func(req *http.Request) (resp *http.Response, err error) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		<-ctx.Done()
		return
	})
	s = NewHTTPSender(HTTPSenderOptions{Client: timeoutMockedHTTPClient})
	if _, err := s.SendWithTimeout(&http.Request{}, time.Millisecond); err == nil {
		t.Fatal("expected error, got nil")
	}
	if err := s.SendJSON(HTTPSendJSONOptions{Timeout: time.Millisecond}); err == nil {
		t.Fatal("expected error, got nil")
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
