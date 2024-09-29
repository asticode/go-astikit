package astikit

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"time"
)

// Copy is a copy with a context
func Copy(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	return io.Copy(dst, NewCtxReader(ctx, src))
}

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }

// NopCloser returns a WriteCloser with a no-op Close method wrapping
// the provided Writer w.
func NopCloser(w io.Writer) io.WriteCloser {
	return nopCloser{w}
}

// CtxReader represents a reader with a context
type CtxReader struct {
	ctx    context.Context
	reader io.Reader
}

// NewCtxReader creates a reader with a context
func NewCtxReader(ctx context.Context, r io.Reader) *CtxReader {
	return &CtxReader{
		ctx:    ctx,
		reader: r,
	}
}

// Read implements the io.Reader interface
func (r *CtxReader) Read(p []byte) (n int, err error) {
	// Check context
	if err = r.ctx.Err(); err != nil {
		return
	}

	// Read
	return r.reader.Read(p)
}

// WriterAdapter represents an object that can adapt a Writer
type WriterAdapter struct {
	buffer *bytes.Buffer
	o      WriterAdapterOptions
}

// WriterAdapterOptions represents WriterAdapter options
type WriterAdapterOptions struct {
	Callback func(i []byte)
	Split    []byte
}

// NewWriterAdapter creates a new WriterAdapter
func NewWriterAdapter(o WriterAdapterOptions) *WriterAdapter {
	return &WriterAdapter{
		buffer: &bytes.Buffer{},
		o:      o,
	}
}

// Close closes the adapter properly
func (w *WriterAdapter) Close() error {
	if w.buffer.Len() > 0 {
		w.write(w.buffer.Bytes())
	}
	return nil
}

// Write implements the io.Writer interface
func (w *WriterAdapter) Write(i []byte) (n int, err error) {
	// Update n to avoid broken pipe error
	defer func() {
		n = len(i)
	}()

	// Split
	if len(w.o.Split) > 0 {
		// Split bytes are not present, write in buffer
		if !bytes.Contains(i, w.o.Split) {
			w.buffer.Write(i)
			return
		}

		// Loop in split items
		items := bytes.Split(i, w.o.Split)
		for i := 0; i < len(items)-1; i++ {
			// If this is the first item, prepend the buffer
			if i == 0 {
				items[i] = append(w.buffer.Bytes(), items[i]...)
				w.buffer.Reset()
			}

			// Write
			w.write(items[i])
		}

		// Add remaining to buffer
		w.buffer.Write(items[len(items)-1])
		return
	}

	// By default, forward the bytes
	w.write(i)
	return
}

func (w *WriterAdapter) write(i []byte) {
	if w.o.Callback != nil {
		w.o.Callback(i)
	}
}

// Piper doesn't block on writes. It will block on reads unless you provide a ReadTimeout
// in which case it will return, after the provided timeout, if no read is available. When closing the
// piper, it will interrupt any ongoing read/future writes and return io.EOF.
// Piper doesn't handle multiple readers at the same time.
type Piper struct {
	buf    [][]byte
	c      *sync.Cond
	closed bool
	o      PiperOptions
	m      sync.Mutex
}

type PiperOptions struct {
	ReadTimeout time.Duration
}

func NewPiper(o PiperOptions) *Piper {
	return &Piper{
		c: sync.NewCond(&sync.Mutex{}),
		o: o,
	}
}

func (p *Piper) Close() error {
	// Update closed
	p.m.Lock()
	if p.closed {
		p.m.Unlock()
		return nil
	}
	p.closed = true
	p.m.Unlock()

	// Signal
	p.c.L.Lock()
	p.c.Signal()
	p.c.L.Unlock()
	return nil
}

func (p *Piper) Read(i []byte) (n int, err error) {
	// Handle read timeout
	var ctx context.Context
	if p.o.ReadTimeout > 0 {
		// Create context
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), p.o.ReadTimeout)
		defer cancel()

		// Watch the context in a goroutine
		go func() {
			// Wait for context to be done
			<-ctx.Done()

			// Context has timed out
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				// Signal
				p.c.L.Lock()
				p.c.Signal()
				p.c.L.Unlock()
			}
		}()
	}

	// Loop
	for {
		// Check context
		if ctx != nil && ctx.Err() != nil {
			return 0, nil
		}

		// Lock
		p.c.L.Lock()
		p.m.Lock()

		// Closed
		if p.closed {
			p.m.Unlock()
			p.c.L.Unlock()
			return 0, io.EOF
		}

		// Get buffer length
		l := len(p.buf)
		p.m.Unlock()

		// Nothing in the buffer, we need to wait
		if l == 0 {
			p.c.Wait()
			p.c.L.Unlock()
			continue
		}
		p.c.L.Unlock()

		// Copy
		p.m.Lock()
		n = len(p.buf[0])
		copy(i, p.buf[0])
		p.buf = p.buf[1:]
		p.m.Unlock()
		return
	}
}

func (p *Piper) Write(i []byte) (n int, err error) {
	// Closed
	p.m.Lock()
	if p.closed {
		p.m.Unlock()
		return 0, io.EOF
	}
	p.m.Unlock()

	// Copy
	b := make([]byte, len(i))
	copy(b, i)

	// Append
	p.m.Lock()
	p.buf = append(p.buf, b)
	p.m.Unlock()

	// Signal
	p.c.L.Lock()
	p.c.Signal()
	p.c.L.Unlock()
	return len(b), nil
}
