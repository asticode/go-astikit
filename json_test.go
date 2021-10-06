package astikit

import "testing"

type jsonA struct {
	A string `json:"a"`
}

type jsonB struct {
	B string `json:"a"`
}

func TestJSONEqual(t *testing.T) {
	if JSONEqual(jsonA{A: "a"}, jsonB{B: "b"}) {
		t.Error("expected false, got true")
	}
	if !JSONEqual(jsonA{A: "a"}, jsonB{B: "a"}) {
		t.Error("expected true, got false")
	}
}
