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
	if !reflect.DeepEqual(o, []string{"bla bla bla ", "bla bla"}) {
		t.Errorf("expected %+v, got %+v", []string{"bla bla bla ", "bla bla"}, o)
	}

	// Close
	w.Close()
	if !reflect.DeepEqual(o, []string{"bla bla bla ", "bla bla", "bla"}) {
		t.Errorf("expected %+v, got %+v", []string{"bla bla bla ", "bla bla", "bla"}, o)
	}
}
