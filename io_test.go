package astikit

import (
	"reflect"
	"testing"
)

func TestWriterAdapter(t *testing.T) {
	// Init
	var o []string
	var w = NewWriterAdapter(WriterAdapterOptions{
		Callback: func(i []byte) {
			o = append(o, string(i))
		},
		Split: []byte("\n"),
	})

	// No Split
	w.Write([]byte("bla bla "))
	if len(o) != 0 {
		t.Errorf("expected %v, got %v", 0, len(o))
	}

	// Multi Split
	w.Write([]byte("bla \nbla bla\nbla"))
	if e := []string{"bla bla bla ", "bla bla"}; !reflect.DeepEqual(o, e) {
		t.Errorf("expected %+v, got %+v", e, o)
	}

	// Close
	w.Close()
	if e := []string{"bla bla bla ", "bla bla", "bla"}; !reflect.DeepEqual(o, e) {
		t.Errorf("expected %+v, got %+v", e, o)
	}
}
