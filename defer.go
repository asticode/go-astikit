package astikit

import (
	"sync"
)

// CloseFunc is a method that closes something
type CloseFunc func() error

// Closer is an object that can close several things
type Closer struct {
	closed bool
	fs     []CloseFunc
	// We need to split into 2 mutexes to allow using .Add() in .Do()
	mc *sync.Mutex // Locks .Close()
	mf *sync.Mutex // Locks fs
}

// NewCloser creates a new closer
func NewCloser() *Closer {
	return &Closer{
		mc: &sync.Mutex{},
		mf: &sync.Mutex{},
	}
}

// Close implements the io.Closer interface
func (c *Closer) Close() error {
	// Lock
	c.mc.Lock()
	defer c.mc.Unlock()

	// Get funcs
	c.mf.Lock()
	fs := c.fs
	c.mf.Unlock()

	// Loop through closers
	err := NewErrors()
	for _, f := range fs {
		err.Add(f())
	}

	// Reset closers
	c.fs = []CloseFunc{}

	// Update attribute
	c.closed = true

	// Return
	if err.IsNil() {
		return nil
	}
	return err
}

// Add adds a close func at the beginning of the list
func (c *Closer) Add(f CloseFunc) {
	// Lock
	c.mf.Lock()
	defer c.mf.Unlock()

	// Append
	c.fs = append([]CloseFunc{f}, c.fs...)
}

// NewChild creates a new child closer
func (c *Closer) NewChild() (child *Closer) {
	child = NewCloser()
	c.Add(child.Close)
	return
}

// Do executes a callback while ensuring :
//   - closer hasn't been closed before
//   - closer can't be closed in between
func (c *Closer) Do(fn func()) {
	// Lock
	c.mc.Lock()
	defer c.mc.Unlock()

	// Closer already closed
	if c.closed {
		return
	}

	// Callback
	fn()
}
