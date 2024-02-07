package astikit

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestSleep(t *testing.T) {
	var ctx, cancel = context.WithCancel(context.Background())
	var err error
	var wg = &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = Sleep(ctx, time.Minute)
	}()
	cancel()
	wg.Wait()
	if e, g := context.Canceled, err; !errors.Is(g, e) {
		t.Fatalf("err should be %s, got %s", e, g)
	}
}

func TestTimestamp(t *testing.T) {
	const j = `{"value":1495290215}`
	v := struct {
		Value Timestamp `json:"value"`
	}{}
	err := json.Unmarshal([]byte(j), &v)
	if err != nil {
		t.Fatalf("err should be nil, got %s", err)
	}
	if e, g := int64(1495290215), v.Value.Unix(); g != e {
		t.Fatalf("timestamp should be %v, got %v", e, g)
	}
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("err should be nil, got %s", err)
	}
	if string(b) != j {
		t.Fatalf("json should be %s, got %s", j, b)
	}
}
