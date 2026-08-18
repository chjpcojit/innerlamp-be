// Package metrics provides registry implementation for managing multiple collectors
package metrics

import (
	"sync"
)

// registryImpl implements the Registry interface
type registryImpl struct {
	mu         sync.RWMutex
	collectors map[string]Collector
	default_   Collector
}

// NewRegistry creates a new metrics registry
func NewRegistry() Registry {
	return &registryImpl{
		collectors: make(map[string]Collector),
	}
}

// Register registers a collector with the given name
func (r *registryImpl) Register(name string, collector Collector) error {
	if name == "" {
		return ErrInvalidConfiguration
	}
	if collector == nil {
		return ErrInvalidConfiguration
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.collectors[name]; exists {
		return ErrCollectorExists
	}

	r.collectors[name] = collector

	// Set as default if first collector
	if r.default_ == nil {
		r.default_ = collector
	}

	return nil
}

// Get retrieves a collector by name
func (r *registryImpl) Get(name string) (Collector, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	collector, exists := r.collectors[name]
	if !exists {
		return nil, ErrCollectorNotFound
	}

	return collector, nil
}

// Unregister removes a collector from the registry
func (r *registryImpl) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	collector, exists := r.collectors[name]
	if !exists {
		return ErrCollectorNotFound
	}

	delete(r.collectors, name)

	// Clear default if it was this collector
	if r.default_ == collector {
		r.default_ = nil
		// Set new default to any remaining collector
		for _, c := range r.collectors {
			r.default_ = c
			break
		}
	}

	return nil
}

// List returns all registered collector names
func (r *registryImpl) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.collectors))
	for name := range r.collectors {
		names = append(names, name)
	}
	return names
}

// Default returns the default collector
func (r *registryImpl) Default() Collector {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.default_
}

// SetDefault sets the default collector
func (r *registryImpl) SetDefault(collector Collector) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.default_ = collector
}

// DefaultRegistry is the global default registry
var DefaultRegistry Registry = NewRegistry()
