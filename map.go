package astikit

import (
	"fmt"
	"sync"
)

// BiMap represents a bidirectional map
type BiMap struct {
	forward map[any]any
	inverse map[any]any
	m       *sync.Mutex
}

// NewBiMap creates a new BiMap
func NewBiMap() *BiMap {
	return &BiMap{
		forward: make(map[any]any),
		inverse: make(map[any]any),
		m:       &sync.Mutex{},
	}
}

func (m *BiMap) get(k any, i map[any]any) (v any, ok bool) {
	m.m.Lock()
	defer m.m.Unlock()
	v, ok = i[k]
	return
}

// Get gets the value in the forward map based on the provided key
func (m *BiMap) Get(k any) (any, bool) { return m.get(k, m.forward) }

// GetInverse gets the value in the inverse map based on the provided key
func (m *BiMap) GetInverse(k any) (any, bool) { return m.get(k, m.inverse) }

// MustGet gets the value in the forward map based on the provided key and panics if key is not found
func (m *BiMap) MustGet(k any) any {
	v, ok := m.get(k, m.forward)
	if !ok {
		panic(fmt.Sprintf("astikit: key %+v not found in foward map", k))
	}
	return v
}

// MustGetInverse gets the value in the inverse map based on the provided key and panics if key is not found
func (m *BiMap) MustGetInverse(k any) any {
	v, ok := m.get(k, m.inverse)
	if !ok {
		panic(fmt.Sprintf("astikit: key %+v not found in inverse map", k))
	}
	return v
}

func (m *BiMap) set(k, v any, f, i map[any]any) *BiMap {
	m.m.Lock()
	defer m.m.Unlock()
	f[k] = v
	i[v] = k
	return m
}

// Set sets the value in the forward and inverse map for the provided forward key
func (m *BiMap) Set(k, v any) *BiMap { return m.set(k, v, m.forward, m.inverse) }

// SetInverse sets the value in the forward and inverse map for the provided inverse key
func (m *BiMap) SetInverse(k, v any) *BiMap { return m.set(k, v, m.inverse, m.forward) }
