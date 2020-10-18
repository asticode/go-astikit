package astikit

import "testing"

func TestBiMap(t *testing.T) {
	m := NewBiMap()
	m.Set(0, 1)
	v, ok := m.Get(0)
	if !ok {
		t.Error("expected true, got false")
	}
	if e, g := 1, v.(int); e != g {
		t.Errorf("expected %d, got %d", e, g)
	}
	_, ok = m.GetInverse(0)
	if ok {
		t.Error("expected false, got true")
	}
	v, ok = m.GetInverse(1)
	if !ok {
		t.Error("expected true, got false")
	}
	if e, g := 0, v.(int); e != g {
		t.Errorf("expected %d, got %d", e, g)
	}
	m.SetInverse(0, 1)
	v, ok = m.GetInverse(0)
	if !ok {
		t.Error("expected true, got false")
	}
	if e, g := 1, v.(int); e != g {
		t.Errorf("expected %d, got %d", e, g)
	}
	testPanic(t, false, func() { m.MustGet(0) })
	testPanic(t, true, func() { m.MustGet(2) })
	testPanic(t, false, func() { m.MustGetInverse(0) })
	testPanic(t, true, func() { m.MustGetInverse(2) })
}

func testPanic(t *testing.T, shouldPanic bool, fn func()) {
	defer func() {
		err := recover()
		if shouldPanic && err == nil {
			t.Error("should have panicked")
		} else if !shouldPanic && err != nil {
			t.Error("should not have panicked")
		}
	}()
	fn()
}
