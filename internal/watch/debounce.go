package watch

import (
	"sync"
	"time"
)

// Debouncer aggregates rapid events and fires once after a quiet period
type Debouncer struct {
	delay    time.Duration
	callback func(path string)
	timers   map[string]*time.Timer
	mu       sync.Mutex
}

// NewDebouncer creates a new debouncer with the specified delay
func NewDebouncer(delay time.Duration, callback func(path string)) *Debouncer {
	return &Debouncer{
		delay:    delay,
		callback: callback,
		timers:   make(map[string]*time.Timer),
	}
}

// Trigger starts or resets the debounce timer for a path
func (d *Debouncer) Trigger(path string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Cancel existing timer for this path
	if timer, exists := d.timers[path]; exists {
		timer.Stop()
	}

	// Create new timer
	d.timers[path] = time.AfterFunc(d.delay, func() {
		d.mu.Lock()
		delete(d.timers, path)
		d.mu.Unlock()

		if d.callback != nil {
			d.callback(path)
		}
	})
}

// Cancel cancels any pending callback for a path
func (d *Debouncer) Cancel(path string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if timer, exists := d.timers[path]; exists {
		timer.Stop()
		delete(d.timers, path)
	}
}

// CancelAll cancels all pending callbacks
func (d *Debouncer) CancelAll() {
	d.mu.Lock()
	defer d.mu.Unlock()

	for path, timer := range d.timers {
		timer.Stop()
		delete(d.timers, path)
	}
}

// Pending returns the number of pending callbacks
func (d *Debouncer) Pending() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.timers)
}

// BatchDebouncer collects multiple events and fires them as a batch
type BatchDebouncer struct {
	delay    time.Duration
	callback func(paths []string)
	timer    *time.Timer
	paths    map[string]struct{}
	mu       sync.Mutex
}

// NewBatchDebouncer creates a debouncer that batches events
func NewBatchDebouncer(delay time.Duration, callback func(paths []string)) *BatchDebouncer {
	return &BatchDebouncer{
		delay:    delay,
		callback: callback,
		paths:    make(map[string]struct{}),
	}
}

// Add adds a path to the current batch
func (bd *BatchDebouncer) Add(path string) {
	bd.mu.Lock()
	defer bd.mu.Unlock()

	// Add path to set
	bd.paths[path] = struct{}{}

	// Reset timer
	if bd.timer != nil {
		bd.timer.Stop()
	}

	bd.timer = time.AfterFunc(bd.delay, func() {
		bd.mu.Lock()
		paths := make([]string, 0, len(bd.paths))
		for p := range bd.paths {
			paths = append(paths, p)
		}
		bd.paths = make(map[string]struct{})
		bd.timer = nil
		bd.mu.Unlock()

		if bd.callback != nil && len(paths) > 0 {
			bd.callback(paths)
		}
	})
}

// Flush immediately fires the callback with current batch
func (bd *BatchDebouncer) Flush() {
	bd.mu.Lock()
	if bd.timer != nil {
		bd.timer.Stop()
		bd.timer = nil
	}
	paths := make([]string, 0, len(bd.paths))
	for p := range bd.paths {
		paths = append(paths, p)
	}
	bd.paths = make(map[string]struct{})
	bd.mu.Unlock()

	if bd.callback != nil && len(paths) > 0 {
		bd.callback(paths)
	}
}

// Cancel cancels the pending batch
func (bd *BatchDebouncer) Cancel() {
	bd.mu.Lock()
	defer bd.mu.Unlock()

	if bd.timer != nil {
		bd.timer.Stop()
		bd.timer = nil
	}
	bd.paths = make(map[string]struct{})
}

// Pending returns the number of paths in the current batch
func (bd *BatchDebouncer) Pending() int {
	bd.mu.Lock()
	defer bd.mu.Unlock()
	return len(bd.paths)
}

// WriteCompleteWaiter waits for a file to finish being written
type WriteCompleteWaiter struct {
	checkInterval time.Duration
	stableTime    time.Duration
}

// NewWriteCompleteWaiter creates a waiter that checks for write completion
func NewWriteCompleteWaiter(checkInterval, stableTime time.Duration) *WriteCompleteWaiter {
	return &WriteCompleteWaiter{
		checkInterval: checkInterval,
		stableTime:    stableTime,
	}
}

// Wait waits for a file to stop changing size
func (w *WriteCompleteWaiter) Wait(path string, getSizeFn func(string) (int64, error)) error {
	var lastSize int64 = -1
	stableStart := time.Time{}

	for {
		size, err := getSizeFn(path)
		if err != nil {
			return err
		}

		if size == lastSize {
			// Size is stable
			if stableStart.IsZero() {
				stableStart = time.Now()
			} else if time.Since(stableStart) >= w.stableTime {
				// File has been stable for long enough
				return nil
			}
		} else {
			// Size changed, reset
			lastSize = size
			stableStart = time.Time{}
		}

		time.Sleep(w.checkInterval)
	}
}
