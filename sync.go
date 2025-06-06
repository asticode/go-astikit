package astikit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Stat names
const (
	StatNameWorkRatio = "astikit.work.ratio"
)

// Chan constants
const (
	// Calling Add() only blocks if the chan has been started and the ctx
	// has not been canceled
	ChanAddStrategyBlockWhenStarted = "block.when.started"
	// Calling Add() never blocks
	ChanAddStrategyNoBlock = "no.block"
	ChanOrderFIFO          = "fifo"
	ChanOrderFILO          = "filo"
)

// Chan is an object capable of executing funcs in a specific order while controlling the conditions
// in which adding new funcs is blocking
// Check out ChanOptions for detailed options
type Chan struct {
	cancel           context.CancelFunc
	c                *sync.Cond
	ctx              context.Context
	fs               []func()
	mc               *sync.Mutex // Locks ctx
	mf               *sync.Mutex // Locks fs
	o                ChanOptions
	running          uint32
	statWorkDuration *AtomicDuration
}

// ChanOptions are Chan options
type ChanOptions struct {
	// Determines the conditions in which Add() blocks. See constants with pattern ChanAddStrategy*
	// Default is ChanAddStrategyNoBlock
	AddStrategy string
	// Order in which the funcs will be processed. See constants with pattern ChanOrder*
	// Default is ChanOrderFIFO
	Order string
	// By default the funcs not yet processed when the context is cancelled are dropped.
	// If "ProcessAll" is true,  ALL funcs are processed even after the context is cancelled.
	// However, no funcs can be added after the context is cancelled
	ProcessAll bool
}

// NewChan creates a new Chan
func NewChan(o ChanOptions) *Chan {
	return &Chan{
		c:                sync.NewCond(&sync.Mutex{}),
		mc:               &sync.Mutex{},
		mf:               &sync.Mutex{},
		o:                o,
		statWorkDuration: NewAtomicDuration(0),
	}
}

// Start starts the chan by looping through functions in the buffer and
// executing them if any, or waiting for a new one otherwise
func (c *Chan) Start(ctx context.Context) {
	// Make sure to start only once
	if atomic.CompareAndSwapUint32(&c.running, 0, 1) {
		// Update status
		defer atomic.StoreUint32(&c.running, 0)

		// Create context
		c.mc.Lock()
		c.ctx, c.cancel = context.WithCancel(ctx)
		d := c.ctx.Done()
		c.mc.Unlock()

		// Handle context
		go func() {
			// Wait for context to be done
			<-d

			// Signal
			c.c.L.Lock()
			c.c.Signal()
			c.c.L.Unlock()
		}()

		// Loop
		for {
			// Lock cond here in case a func is added between retrieving l and doing the if on it
			c.c.L.Lock()

			// Get number of funcs in buffer
			c.mf.Lock()
			l := len(c.fs)
			c.mf.Unlock()

			// Only return if context has been cancelled and:
			//   - the user wants to drop funcs that has not yet been processed
			//   - the buffer is empty otherwise
			c.mc.Lock()
			if c.ctx.Err() != nil && (!c.o.ProcessAll || l == 0) {
				c.mc.Unlock()
				c.c.L.Unlock()
				return
			}
			c.mc.Unlock()

			// No funcs in buffer
			if l == 0 {
				c.c.Wait()
				c.c.L.Unlock()
				continue
			}
			c.c.L.Unlock()

			// Get first func
			c.mf.Lock()
			fn := c.fs[0]
			c.mf.Unlock()

			// Execute func
			n := time.Now()
			fn()
			c.statWorkDuration.Add(time.Since(n))

			// Remove first func
			c.mf.Lock()
			c.fs = c.fs[1:]
			c.mf.Unlock()
		}
	}
}

// Stop stops the chan
func (c *Chan) Stop() {
	c.mc.Lock()
	if c.cancel != nil {
		c.cancel()
	}
	c.mc.Unlock()
}

// Add adds a new item to the chan
func (c *Chan) Add(i func()) {
	// Check context
	c.mc.Lock()
	if c.ctx != nil && c.ctx.Err() != nil {
		c.mc.Unlock()
		return
	}
	c.mc.Unlock()

	// Wrap the function
	var fn func()
	var wg *sync.WaitGroup
	if c.o.AddStrategy == ChanAddStrategyBlockWhenStarted {
		wg = &sync.WaitGroup{}
		wg.Add(1)
		fn = func() {
			defer wg.Done()
			i()
		}
	} else {
		fn = i
	}

	// Add func to buffer
	c.mf.Lock()
	if c.o.Order == ChanOrderFILO {
		c.fs = append([]func(){fn}, c.fs...)
	} else {
		c.fs = append(c.fs, fn)
	}
	c.mf.Unlock()

	// Signal
	c.c.L.Lock()
	c.c.Signal()
	c.c.L.Unlock()

	// Wait
	if wg != nil {
		wg.Wait()
	}
}

// Reset resets the chan
func (c *Chan) Reset() {
	c.mf.Lock()
	defer c.mf.Unlock()
	c.fs = []func(){}
}

// ChanStats represents the chan stats
type ChanStats struct {
	WorkDuration time.Duration
}

// Stats returns the chan stats
func (c *Chan) Stats() ChanStats {
	return ChanStats{WorkDuration: c.statWorkDuration.Duration()}
}

// StatOptions returns the chan stat options
func (c *Chan) StatOptions() []StatOptions {
	return []StatOptions{
		{
			Metadata: &StatMetadata{
				Description: "Percentage of time doing work",
				Label:       "Work ratio",
				Name:        StatNameWorkRatio,
				Unit:        "%",
			},
			Valuer: NewAtomicDurationPercentageStat(c.statWorkDuration),
		},
	}
}

// BufferPool represents a *bytes.Buffer pool
type BufferPool struct {
	bp *sync.Pool
}

// NewBufferPool creates a new BufferPool
func NewBufferPool() *BufferPool {
	return &BufferPool{bp: &sync.Pool{New: func() any { return &bytes.Buffer{} }}}
}

// New creates a new BufferPoolItem
func (p *BufferPool) New() *BufferPoolItem {
	return newBufferPoolItem(p.bp.Get().(*bytes.Buffer), p.bp)
}

// BufferPoolItem represents a BufferPool item
type BufferPoolItem struct {
	*bytes.Buffer
	bp *sync.Pool
}

func newBufferPoolItem(b *bytes.Buffer, bp *sync.Pool) *BufferPoolItem {
	return &BufferPoolItem{
		Buffer: b,
		bp:     bp,
	}
}

// Close implements the io.Closer interface
func (i *BufferPoolItem) Close() error {
	i.Reset()
	i.bp.Put(i.Buffer)
	return nil
}

// GoroutineLimiter is an object capable of doing several things in parallel while maintaining the
// max number of things running in parallel under a threshold
type GoroutineLimiter struct {
	busy   int
	c      *sync.Cond
	ctx    context.Context
	cancel context.CancelFunc
	o      GoroutineLimiterOptions
}

// GoroutineLimiterOptions represents GoroutineLimiter options
type GoroutineLimiterOptions struct {
	Max int
}

// NewGoroutineLimiter creates a new GoroutineLimiter
func NewGoroutineLimiter(o GoroutineLimiterOptions) (l *GoroutineLimiter) {
	l = &GoroutineLimiter{
		c: sync.NewCond(&sync.Mutex{}),
		o: o,
	}
	if l.o.Max <= 0 {
		l.o.Max = 1
	}
	l.ctx, l.cancel = context.WithCancel(context.Background())
	go l.handleCtx()
	return
}

// Close closes the limiter properly
func (l *GoroutineLimiter) Close() error {
	l.cancel()
	return nil
}

func (l *GoroutineLimiter) handleCtx() {
	<-l.ctx.Done()
	l.c.L.Lock()
	l.c.Broadcast()
	l.c.L.Unlock()
}

// GoroutineLimiterFunc is a GoroutineLimiter func
type GoroutineLimiterFunc func()

// Do executes custom work in a goroutine
func (l *GoroutineLimiter) Do(fn GoroutineLimiterFunc) (err error) {
	// Check context in case the limiter has already been closed
	if err = l.ctx.Err(); err != nil {
		return
	}

	// Lock
	l.c.L.Lock()

	// Wait for a goroutine to be available
	for l.busy >= l.o.Max {
		l.c.Wait()
	}

	// Check context in case the limiter has been closed while waiting
	if err = l.ctx.Err(); err != nil {
		return
	}

	// Increment
	l.busy++

	// Unlock
	l.c.L.Unlock()

	// Execute in a goroutine
	go func() {
		// Decrement
		defer func() {
			l.c.L.Lock()
			l.busy--
			l.c.Signal()
			l.c.L.Unlock()
		}()

		// Execute
		fn()
	}()
	return
}

// Eventer represents an object that can dispatch simple events (name + payload)
type Eventer struct {
	c  *Chan
	hs map[string][]EventerHandler
	mh *sync.Mutex
}

// EventerOptions represents Eventer options
type EventerOptions struct {
	Chan ChanOptions
}

// EventerHandler represents a function that can handle the payload of an event
type EventerHandler func(payload any)

// NewEventer creates a new eventer
func NewEventer(o EventerOptions) *Eventer {
	return &Eventer{
		c:  NewChan(o.Chan),
		hs: make(map[string][]EventerHandler),
		mh: &sync.Mutex{},
	}
}

// On adds an handler for a specific name
func (e *Eventer) On(name string, h EventerHandler) {
	// Lock
	e.mh.Lock()
	defer e.mh.Unlock()

	// Add handler
	e.hs[name] = append(e.hs[name], h)
}

// Dispatch dispatches a payload for a specific name
func (e *Eventer) Dispatch(name string, payload any) {
	// Lock
	e.mh.Lock()
	defer e.mh.Unlock()

	// No handlers
	hs, ok := e.hs[name]
	if !ok {
		return
	}

	// Loop through handlers
	for _, h := range hs {
		func(h EventerHandler) {
			// Add to chan
			e.c.Add(func() {
				h(payload)
			})
		}(h)
	}
}

// Start starts the eventer. It is blocking
func (e *Eventer) Start(ctx context.Context) {
	e.c.Start(ctx)
}

// Stop stops the eventer
func (e *Eventer) Stop() {
	e.c.Stop()
}

// Reset resets the eventer
func (e *Eventer) Reset() {
	e.c.Reset()
}

// DebugMutex represents a rwmutex capable of logging its actions to ease deadlock debugging
type DebugMutex struct {
	l               CompleteLogger
	lastCaller      string
	lastCallerMutex *sync.Mutex
	ll              LoggerLevel
	m               *sync.RWMutex
	name            string
	timeout         time.Duration
}

// DebugMutexOpt represents a debug mutex option
type DebugMutexOpt func(m *DebugMutex)

// DebugMutexWithLockLogging allows logging all mutex locks
func DebugMutexWithLockLogging(ll LoggerLevel) DebugMutexOpt {
	return func(m *DebugMutex) {
		m.ll = ll
	}
}

// DebugMutexWithDeadlockDetection allows detecting deadlock for all mutex locks
func DebugMutexWithDeadlockDetection(timeout time.Duration) DebugMutexOpt {
	return func(m *DebugMutex) {
		m.timeout = timeout
	}
}

// NewDebugMutex creates a new debug mutex
func NewDebugMutex(name string, l StdLogger, opts ...DebugMutexOpt) *DebugMutex {
	m := &DebugMutex{
		l:               AdaptStdLogger(l),
		lastCallerMutex: &sync.Mutex{},
		ll:              LoggerLevelDebug - 1,
		m:               &sync.RWMutex{},
		name:            name,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *DebugMutex) caller() (o string) {
	if _, file, line, ok := runtime.Caller(2); ok {
		o = fmt.Sprintf("%s:%d", file, line)
	}
	return
}

func (m *DebugMutex) log(fmt string, args ...any) {
	if m.ll < LoggerLevelDebug {
		return
	}
	m.l.Writef(m.ll, fmt, args...)
}

func (m *DebugMutex) watchTimeout(caller string, fn func()) {
	if m.timeout <= 0 {
		fn()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	go func() {
		<-ctx.Done()
		if err := ctx.Err(); err != nil && errors.Is(err, context.DeadlineExceeded) {
			m.lastCallerMutex.Lock()
			lastCaller := m.lastCaller
			m.lastCallerMutex.Unlock()
			m.l.Errorf("astikit: %s mutex timed out at %s with last caller at %s", m.name, caller, lastCaller)
		}
	}()

	fn()
}

// Lock write locks the mutex
func (m *DebugMutex) Lock() {
	c := m.caller()
	m.log("astikit: requesting lock for %s at %s", m.name, c)
	m.watchTimeout(c, m.m.Lock)
	m.log("astikit: lock acquired for %s at %s", m.name, c)
	m.lastCallerMutex.Lock()
	m.lastCaller = c
	m.lastCallerMutex.Unlock()
}

// Unlock write unlocks the mutex
func (m *DebugMutex) Unlock() {
	m.m.Unlock()
	m.log("astikit: unlock executed for %s", m.name)
}

// RLock read locks the mutex
func (m *DebugMutex) RLock() {
	c := m.caller()
	m.log("astikit: requesting rlock for %s at %s", m.name, c)
	m.watchTimeout(c, m.m.RLock)
	m.log("astikit: rlock acquired for %s at %s", m.name, c)
	m.lastCallerMutex.Lock()
	m.lastCaller = c
	m.lastCallerMutex.Unlock()
}

// RUnlock read unlocks the mutex
func (m *DebugMutex) RUnlock() {
	m.m.RUnlock()
	m.log("astikit: unlock executed for %s", m.name)
}

type AtomicDuration struct {
	d time.Duration
	m *sync.Mutex
}

func NewAtomicDuration(d time.Duration) *AtomicDuration {
	return &AtomicDuration{
		d: d,
		m: &sync.Mutex{},
	}
}

func (d *AtomicDuration) Add(delta time.Duration) {
	d.m.Lock()
	defer d.m.Unlock()
	d.d += delta
}

func (d *AtomicDuration) Duration() time.Duration {
	d.m.Lock()
	defer d.m.Unlock()
	return d.d
}

// FIFOMutex is a mutex guaranteeing FIFO order
type FIFOMutex struct {
	busy    bool
	m       sync.Mutex // Locks busy and waiting
	waiting []*sync.Cond
}

func (m *FIFOMutex) Lock() {
	// No need to wait
	m.m.Lock()
	if !m.busy {
		m.busy = true
		m.m.Unlock()
		return
	}

	// Create cond
	c := sync.NewCond(&sync.Mutex{})

	// Make sure to lock cond when waiting mutex is still held
	c.L.Lock()

	// Add to waiting queue
	m.waiting = append(m.waiting, c)
	m.m.Unlock()

	// Wait
	c.Wait()
}

func (m *FIFOMutex) Unlock() {
	// Lock
	m.m.Lock()
	defer m.m.Unlock()

	// Waiting queue is empty
	if len(m.waiting) == 0 {
		m.busy = false
		return
	}

	// Signal and remove first item in waiting queue
	m.waiting[0].L.Lock()
	m.waiting[0].Signal()
	m.waiting[0].L.Unlock()
	m.waiting = m.waiting[1:]
}

// BufferedBatcher is a Chan-like object that:
//   - processes all added items in the provided callback as a batch so that they're all processed together
//   - doesn't block when adding an item while a batch is being processed but add it to the next batch
//   - if an item is added several times to the same batch, it will be processed only once in the next batch
type BufferedBatcher struct {
	batch   map[any]bool // Locked by c's mutex
	c       *sync.Cond
	cancel  context.CancelFunc
	ctx     context.Context
	mc      sync.Mutex // Locks cancel and ctx
	onBatch BufferedBatcherOnBatchFunc
}

type BufferedBatcherOnBatchFunc func(ctx context.Context, batch []any)

type BufferedBatcherOptions struct {
	OnBatch BufferedBatcherOnBatchFunc
}

func NewBufferedBatcher(o BufferedBatcherOptions) *BufferedBatcher {
	return &BufferedBatcher{
		batch:   make(map[any]bool),
		c:       sync.NewCond(&sync.Mutex{}),
		onBatch: o.OnBatch,
	}
}

func (bb *BufferedBatcher) Start(ctx context.Context) {
	// Already running
	bb.mc.Lock()
	if bb.ctx != nil && bb.ctx.Err() == nil {
		bb.mc.Unlock()
		return
	}

	// Create context
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Store context
	bb.ctx = ctx
	bb.cancel = cancel
	bb.mc.Unlock()

	// Handle context
	go func() {
		// Wait for context to be done
		<-ctx.Done()

		// Signal
		bb.c.L.Lock()
		bb.c.Signal()
		bb.c.L.Unlock()
	}()

	// Loop
	for {
		// Context has been canceled
		if ctx.Err() != nil {
			return
		}

		// Wait for batch
		bb.c.L.Lock()
		if len(bb.batch) == 0 {
			bb.c.Wait()
			bb.c.L.Unlock()
			continue
		}

		// Copy batch into a slice
		var batch []any
		for i := range bb.batch {
			batch = append(batch, i)
		}

		// Reset batch
		bb.batch = map[any]bool{}

		// Unlock
		bb.c.L.Unlock()

		// Callback
		bb.onBatch(ctx, batch)
	}
}

func (bb *BufferedBatcher) Add(i any) {
	// Lock
	bb.c.L.Lock()
	defer bb.c.L.Unlock()

	// Store
	bb.batch[i] = true

	// Signal
	bb.c.Signal()
}

func (bb *BufferedBatcher) Stop() {
	// Lock
	bb.mc.Lock()
	defer bb.mc.Unlock()

	// Not running
	if bb.ctx == nil {
		return
	}

	// Cancel
	bb.cancel()

	// Reset context
	bb.ctx = nil
	bb.cancel = nil
}
