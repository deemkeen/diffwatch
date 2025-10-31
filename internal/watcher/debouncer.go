package watcher

import (
	"sync"
	"time"
)

// Debouncer batches rapid successive events for the same file
type Debouncer struct {
	delay    time.Duration
	timers   map[string]*time.Timer
	mu       sync.Mutex
	stopChan chan struct{}
}

// NewDebouncer creates a new debouncer with the given delay
func NewDebouncer(delay time.Duration) *Debouncer {
	return &Debouncer{
		delay:    delay,
		timers:   make(map[string]*time.Timer),
		stopChan: make(chan struct{}),
	}
}

// Add adds an event for debouncing. The callback will be called after
// the delay period if no new events for the same key arrive
func (d *Debouncer) Add(key string, callback func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Cancel existing timer if present
	if timer, exists := d.timers[key]; exists {
		timer.Stop()
	}

	// Create new timer
	d.timers[key] = time.AfterFunc(d.delay, func() {
		d.mu.Lock()
		delete(d.timers, key)
		d.mu.Unlock()

		callback()
	})
}

// Stop stops all timers and cleans up
func (d *Debouncer) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	close(d.stopChan)

	for _, timer := range d.timers {
		timer.Stop()
	}
	d.timers = make(map[string]*time.Timer)
}
