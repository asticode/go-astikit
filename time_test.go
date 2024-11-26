package astikit

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
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

func isAbsoluteTime(t time.Time) bool {
	return t.Year() >= time.Now().Year()-1
}

func TestNow(t *testing.T) {
	if g := Now(); !isAbsoluteTime(Now()) {
		t.Fatalf("expected %s to be an absolute time", g)
	}
	var count int64
	m := MockNow(func() time.Time {
		count++
		return time.Unix(count, 0)
	})
	if e, g := time.Unix(1, 0), Now(); !reflect.DeepEqual(e, g) {
		t.Fatalf("expected %s, got %s", e, g)
	}
	m.Close()
	if g := Now(); !isAbsoluteTime(Now()) {
		t.Fatalf("expected %s to be an absolute time", g)
	}
}
