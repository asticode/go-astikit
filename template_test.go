package astikit

import (
	"bytes"
	"testing"
)

func TestTemplater(t *testing.T) {
	tp := NewTemplater()
	if err := tp.AddLayoutsFromDir("testdata/template/layouts", ".html"); err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	if len(tp.layouts) != 2 {
		t.Errorf("expected %v, got %v", 2, len(tp.layouts))
	}
	if err := tp.AddTemplatesFromDir("testdata/template/templates", ".html"); err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	if len(tp.templates) != 2 {
		t.Errorf("expected %v, got %v", 2, len(tp.templates))
	}
	tp.DelTemplate("/dir/template2.html")
	if len(tp.templates) != 1 {
		t.Errorf("expected %v, got %v", 1, len(tp.templates))
	}
	v, ok := tp.Template("/template1.html")
	if !ok {
		t.Error("no template found")
	}
	w := &bytes.Buffer{}
	if err := v.Execute(w, nil); err != nil {
		t.Errorf("expected no error, got %+v", err)
	}
	if w.String() != "Layout - Template" {
		t.Errorf("expected %s, got %s", "Layout - Template", w.String())
	}
}
