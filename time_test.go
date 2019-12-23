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
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err should be %s, got %s", context.Canceled, err)
	}
}

func TestTimestamp(t *testing.T) {
	const j = `{"value":1495290215}`
	v := struct {
		Value Timestamp `json:"value"`
	}{}
	err := json.Unmarshal([]byte(j), &v)
	if err != nil {
		t.Errorf("err should be nil, got %s", err)
	}
	if v.Value.Unix() != 1495290215 {
		t.Errorf("timestamp should be %v, got %v", 1495290215, v.Value.Unix())
	}
	b, err := json.Marshal(v)
	if err != nil {
		t.Errorf("err should be nil, got %s", err)
	}
	if string(b) != j {
		t.Errorf("json should be %s, got %s", j, b)
	}
}
