package astikit

import (
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"testing"
	"time"
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

func TestPiper(t *testing.T) {
	p1 := NewPiper(PiperOptions{})
	defer p1.Close()

	// Piper shouldn't block on write
	w := []byte("test")
	ctx1, cancel1 := context.WithTimeout(context.Background(), time.Second)
	defer cancel1()
	var n int
	var err error
	go func() {
		defer cancel1()
		n, err = p1.Write(w)
	}()
	<-ctx1.Done()
	if errCtx := ctx1.Err(); errors.Is(errCtx, context.DeadlineExceeded) {
		t.Fatalf("expected no deadline exceeded error, got %+v", errCtx)
	}
	if err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	if e, g := 4, n; e != g {
		t.Fatalf("expected %d, got %d", e, g)
	}
	r := make([]byte, 10)
	n, err = p1.Read(r)
	if err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	if e, g := 4, n; e != g {
		t.Fatalf("expected %d, got %d", e, g)
	}
	if e, g := w, r[:n]; !bytes.Equal(e, g) {
		t.Fatalf("expected %s, got %s", e, g)
	}

	// Piper should block on read unless write or piper is closed
	ctx2, cancel2 := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel2()
	ctx3, cancel3 := context.WithTimeout(context.Background(), time.Second)
	defer cancel3()
	r = make([]byte, 10)
	go func() {
		defer cancel2()
		defer cancel3()
		n, err = p1.Read(r)
	}()
	<-ctx2.Done()
	if errCtx := ctx2.Err(); !errors.Is(errCtx, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded error, got %+v", errCtx)
	}
	_, errWrite := p1.Write(w)
	if errWrite != nil {
		t.Fatalf("expected no error, got %+v", errWrite)
	}
	<-ctx3.Done()
	if errCtx := ctx3.Err(); errors.Is(errCtx, context.DeadlineExceeded) {
		t.Fatalf("expected no deadline exceeded error, got %+v", errCtx)
	}
	if err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	if e, g := 4, n; e != g {
		t.Fatalf("expected %d, got %d", e, g)
	}
	if e, g := w, r[:n]; !bytes.Equal(e, g) {
		t.Fatalf("expected %s, got %s", e, g)
	}
	ctx4, cancel4 := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel4()
	ctx5, cancel5 := context.WithTimeout(context.Background(), time.Second)
	defer cancel5()
	go func() {
		defer cancel4()
		defer cancel5()
		_, err = p1.Read(r)
	}()
	<-ctx4.Done()
	if errCtx := ctx4.Err(); !errors.Is(errCtx, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded error, got %+v", errCtx)
	}
	p1.Close()
	<-ctx5.Done()
	if errCtx := ctx5.Err(); errors.Is(errCtx, context.DeadlineExceeded) {
		t.Fatalf("expected no deadline exceeded error, got %+v", errCtx)
	}
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF error, got %+v", err)
	}
	_, err = p1.Write(w)
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF error, got %+v", err)
	}

	// Piper should timeout on read if a read timeout is provided
	p2 := NewPiper(PiperOptions{ReadTimeout: time.Millisecond})
	defer p2.Close()
	ctx6, cancel6 := context.WithTimeout(context.Background(), time.Second)
	defer cancel6()
	go func() {
		defer cancel6()
		_, err = p2.Read(r)
	}()
	<-ctx6.Done()
	if errCtx := ctx6.Err(); errors.Is(errCtx, context.DeadlineExceeded) {
		t.Fatalf("expected no deadline exceeded error, got %+v", errCtx)
	}
	if err != nil {
		t.Fatalf("expected nil, got %+v", err)
	}
}
