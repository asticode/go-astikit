package astikit

import (
	"testing"
	"time"
)

func TestLimiter(t *testing.T) {
	var l = NewLimiter()
	defer l.Close()
	l.Add("test", 2, time.Second)
	b, ok := l.Bucket("test")
	if !ok {
		t.Fatal("no bucket found")
	}
	defer b.Close()
	if !b.Inc() {
		t.Fatalf("got false, expected true")
	}
	if !b.Inc() {
		t.Fatalf("got false, expected true")
	}
	if b.Inc() {
		t.Fatalf("got true, expected false")
	}
}
