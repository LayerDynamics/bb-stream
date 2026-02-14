package progress

import (
	"io"
	"sync"
)

// Callback is a function that receives progress updates
type Callback func(bytesTransferred, totalBytes int64)

// Reader wraps an io.Reader and reports progress
type Reader struct {
	reader      io.Reader
	total       int64
	transferred int64
	callback    Callback
	mu          sync.Mutex
}

// NewReader creates a progress-tracking reader
func NewReader(r io.Reader, total int64, callback Callback) *Reader {
	return &Reader{
		reader:   r,
		total:    total,
		callback: callback,
	}
}

// Read implements io.Reader
func (pr *Reader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 {
		pr.mu.Lock()
		pr.transferred += int64(n)
		transferred := pr.transferred
		pr.mu.Unlock()

		if pr.callback != nil {
			pr.callback(transferred, pr.total)
		}
	}
	return n, err
}

// Writer wraps an io.Writer and reports progress
type Writer struct {
	writer      io.Writer
	total       int64
	transferred int64
	callback    Callback
	mu          sync.Mutex
}

// NewWriter creates a progress-tracking writer
func NewWriter(w io.Writer, total int64, callback Callback) *Writer {
	return &Writer{
		writer:   w,
		total:    total,
		callback: callback,
	}
}

// Write implements io.Writer
func (pw *Writer) Write(p []byte) (int, error) {
	n, err := pw.writer.Write(p)
	if n > 0 {
		pw.mu.Lock()
		pw.transferred += int64(n)
		transferred := pw.transferred
		pw.mu.Unlock()

		if pw.callback != nil {
			pw.callback(transferred, pw.total)
		}
	}
	return n, err
}

// Tracker provides a simple way to track progress
type Tracker struct {
	Total       int64
	Transferred int64
	mu          sync.Mutex
	callbacks   []Callback
}

// NewTracker creates a new progress tracker
func NewTracker(total int64) *Tracker {
	return &Tracker{
		Total: total,
	}
}

// Add registers a callback
func (t *Tracker) Add(callback Callback) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.callbacks = append(t.callbacks, callback)
}

// Update updates the progress and notifies callbacks
func (t *Tracker) Update(transferred int64) {
	t.mu.Lock()
	t.Transferred = transferred
	callbacks := make([]Callback, len(t.callbacks))
	copy(callbacks, t.callbacks)
	t.mu.Unlock()

	for _, cb := range callbacks {
		cb(transferred, t.Total)
	}
}

// Increment adds to the transferred count
func (t *Tracker) Increment(n int64) {
	t.mu.Lock()
	t.Transferred += n
	transferred := t.Transferred
	callbacks := make([]Callback, len(t.callbacks))
	copy(callbacks, t.callbacks)
	t.mu.Unlock()

	for _, cb := range callbacks {
		cb(transferred, t.Total)
	}
}

// Percent returns the completion percentage
func (t *Tracker) Percent() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.Total == 0 {
		return 0
	}
	return float64(t.Transferred) / float64(t.Total) * 100
}
