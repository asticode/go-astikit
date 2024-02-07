package astikit

import (
	"bytes"
	"testing"
)

func TestTemplater(t *testing.T) {
	tp := NewTemplater()
	if err := tp.AddLayoutsFromDir("testdata/template/layouts", ".html"); err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	if e, g := 2, len(tp.layouts); e != g {
		t.Fatalf("expected %v, got %v", e, g)
	}
	if err := tp.AddTemplatesFromDir("testdata/template/templates", ".html"); err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	if e, g := 2, len(tp.templates); e != g {
		t.Fatalf("expected %v, got %v", e, g)
	}
	tp.DelTemplate("/dir/template2.html")
	if e, g := 1, len(tp.templates); e != g {
		t.Fatalf("expected %v, got %v", e, g)
	}
	v, ok := tp.Template("/template1.html")
	if !ok {
		t.Fatal("no template found")
	}
	w := &bytes.Buffer{}
	if err := v.Execute(w, nil); err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	if e, g := "Layout - Template", w.String(); g != e {
		t.Fatalf("expected %s, got %s", e, g)
	}
}
