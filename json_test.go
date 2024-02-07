package astikit

import "testing"

type jsonA struct {
	A string `json:"a"`
}

type jsonB struct {
	B string `json:"a"`
}

func TestJSONClone(t *testing.T) {
	a := jsonA{A: "a"}
	b := &jsonB{}
	err := JSONClone(a, b)
	if err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
	if !JSONEqual(a, b) {
		t.Fatal("expected true, got false")
	}
}

func TestJSONEqual(t *testing.T) {
	if JSONEqual(jsonA{A: "a"}, jsonB{B: "b"}) {
		t.Fatal("expected false, got true")
	}
	if !JSONEqual(jsonA{A: "a"}, jsonB{B: "a"}) {
		t.Fatal("expected true, got false")
	}
}
