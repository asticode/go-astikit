package astikit

import (
	"net"
	"net/http"
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
			w.Stop()
			time.Sleep(time.Millisecond)
			i++
		}),
	})
	go func() {
		c := &http.Client{}
		r, _ := http.NewRequest(http.MethodGet, "http://"+ln.Addr().String(), nil)
		c.Do(r)
	}()
	w.Wait()
	if e := 1; i != e {
		t.Errorf("expected %+v, got %+v", e, i)
	}
}

type mockedHTTPClient func(req *http.Request) (*http.Response, error)

func (c mockedHTTPClient) Do(req *http.Request) (*http.Response, error) { return c(req) }

type mockedNetError struct{ temporary bool }

func (err mockedNetError) Error() string   { return "" }
func (err mockedNetError) Timeout() bool   { return false }
func (err mockedNetError) Temporary() bool { return err.temporary }

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
	_, err := s.Send(&http.Request{})
	if err == nil {
		t.Error("expected error, got nil")
	}
	if e := 4; c != e {
		t.Errorf("expected %v, got %v", e, c)
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
				err = mockedNetError{temporary: true}
			default:
				// No retrying
				resp = &http.Response{StatusCode: http.StatusBadRequest}
			}
			return
		}),
		RetryMax: 3,
	})
	_, err = s.Send(&http.Request{})
	if err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	if e := 3; c != e {
		t.Errorf("expected %v, got %v", e, c)
	}

}
