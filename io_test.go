package astikit

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestCopy(t *testing.T) {
	// Context canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r, w := bytes.NewBuffer([]byte("bla bla bla")), &bytes.Buffer{}
	n, err := Copy(ctx, w, r)
	if e := int64(0); n != e {
		t.Fatalf("expected %v, got %v", e, n)
	}
	if e := context.Canceled; !errors.Is(err, e) {
		t.Fatalf("error should be %+v, got %+v", e, err)
	}

	// Default
	n, err = Copy(context.Background(), w, r)
	if e := int64(11); n != e {
		t.Fatalf("expected %v, got %v", e, n)
	}
	if err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
}

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
	w.Write([]byte("bla bla ")) //nolint:errcheck
	if len(o) != 0 {
		t.Fatalf("expected %v, got %v", 0, len(o))
	}

	// Multi Split
	w.Write([]byte("bla \nbla bla\nbla")) //nolint:errcheck
	if e := []string{"bla bla bla ", "bla bla"}; !reflect.DeepEqual(o, e) {
		t.Fatalf("expected %+v, got %+v", e, o)
	}

	// Close
	w.Close()
	if e := []string{"bla bla bla ", "bla bla", "bla"}; !reflect.DeepEqual(o, e) {
		t.Fatalf("expected %+v, got %+v", e, o)
	}
}
