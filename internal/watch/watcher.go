package watch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/ryanoboyle/bb-stream/internal/b2"
)

// Event represents a file system event
type Event struct {
	Path      string
	Op        Operation
	Timestamp time.Time
}

// Operation represents the type of file system operation
type Operation int

const (
	Create Operation = iota
	Write
	Remove
	Rename
)

func (op Operation) String() string {
	switch op {
	case Create:
		return "create"
	case Write:
		return "write"
	case Remove:
		return "remove"
	case Rename:
		return "rename"
	default:
		return "unknown"
	}
}

// WatcherOptions configures the watcher
type WatcherOptions struct {
	DebounceDelay   time.Duration
	IgnorePatterns  []string
	IncludePatterns []string
	Recursive       bool
	OnEvent         func(Event)
	OnError         func(error)
}

// DefaultWatcherOptions returns sensible defaults
func DefaultWatcherOptions() *WatcherOptions {
	return &WatcherOptions{
		DebounceDelay: 500 * time.Millisecond,
		IgnorePatterns: []string{
			".git",
			".DS_Store",
			"node_modules",
			"__pycache__",
			"*.swp",
			"*.tmp",
			"*~",
		},
		Recursive: true,
	}
}

// Watcher watches a directory for changes
type Watcher struct {
	watcher   *fsnotify.Watcher
	opts      *WatcherOptions
	debouncer *Debouncer
	watching  map[string]struct{}
	mu        sync.RWMutex
	done      chan struct{}
}

// NewWatcher creates a new file system watcher
func NewWatcher(opts *WatcherOptions) (*Watcher, error) {
	if opts == nil {
		opts = DefaultWatcherOptions()
	}

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	w := &Watcher{
		watcher:  fsWatcher,
		opts:     opts,
		watching: make(map[string]struct{}),
		done:     make(chan struct{}),
	}

	// Set up debouncer
	w.debouncer = NewDebouncer(opts.DebounceDelay, func(path string) {
		if w.opts.OnEvent != nil {
			w.opts.OnEvent(Event{
				Path:      path,
				Op:        Write,
				Timestamp: time.Now(),
			})
		}
	})

	return w, nil
}

// Watch starts watching a directory
func (w *Watcher) Watch(ctx context.Context, path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Add the root directory
	if err := w.addPath(absPath); err != nil {
		return err
	}

	// If recursive, add all subdirectories
	if w.opts.Recursive {
		err := filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors
			}
			if info.IsDir() && !w.shouldIgnore(path) {
				return w.addPath(path)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to walk directory: %w", err)
		}
	}

	// Start event loop
	go w.eventLoop(ctx)

	return nil
}

// addPath adds a path to the watch list
func (w *Watcher) addPath(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.watching[path]; exists {
		return nil // Already watching
	}

	if err := w.watcher.Add(path); err != nil {
		return fmt.Errorf("failed to watch %s: %w", path, err)
	}

	w.watching[path] = struct{}{}
	return nil
}

// removePath removes a path from the watch list
func (w *Watcher) removePath(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.watching[path]; !exists {
		return
	}

	w.watcher.Remove(path)
	delete(w.watching, path)
}

// eventLoop processes file system events
func (w *Watcher) eventLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.handleEvent(event)
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			if w.opts.OnError != nil {
				w.opts.OnError(err)
			}
		}
	}
}

// handleEvent processes a single file system event
func (w *Watcher) handleEvent(event fsnotify.Event) {
	path := event.Name

	// Check if we should ignore this path
	if w.shouldIgnore(path) {
		return
	}

	// Check if file matches include patterns (if specified)
	if len(w.opts.IncludePatterns) > 0 && !w.shouldInclude(path) {
		return
	}

	// Handle directory creation - add to watch list
	if event.Op&fsnotify.Create != 0 {
		info, err := os.Stat(path)
		if err == nil && info.IsDir() && w.opts.Recursive {
			w.addPath(path)
		}
	}

	// Handle directory removal - remove from watch list
	if event.Op&fsnotify.Remove != 0 {
		w.removePath(path)
	}

	// Determine operation type
	var op Operation
	switch {
	case event.Op&fsnotify.Create != 0:
		op = Create
	case event.Op&fsnotify.Write != 0:
		op = Write
	case event.Op&fsnotify.Remove != 0:
		op = Remove
	case event.Op&fsnotify.Rename != 0:
		op = Rename
	default:
		return // Ignore other operations
	}

	// For write events, debounce to wait for file to finish writing
	if op == Write || op == Create {
		w.debouncer.Trigger(path)
	} else if w.opts.OnEvent != nil {
		w.opts.OnEvent(Event{
			Path:      path,
			Op:        op,
			Timestamp: time.Now(),
		})
	}
}

// shouldIgnore checks if a path should be ignored
func (w *Watcher) shouldIgnore(path string) bool {
	for _, pattern := range w.opts.IgnorePatterns {
		// Check full path
		if strings.Contains(path, pattern) {
			return true
		}
		// Check filename
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
	}
	return false
}

// shouldInclude checks if a path matches include patterns
func (w *Watcher) shouldInclude(path string) bool {
	for _, pattern := range w.opts.IncludePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
	}
	return false
}

// Stop stops the watcher
func (w *Watcher) Stop() {
	close(w.done)
	w.watcher.Close()
	w.debouncer.CancelAll()
}

// Paths returns the currently watched paths
func (w *Watcher) Paths() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	paths := make([]string, 0, len(w.watching))
	for path := range w.watching {
		paths = append(paths, path)
	}
	return paths
}

// AutoUploader watches a directory and uploads changed files to B2
type AutoUploader struct {
	client     *b2.Client
	watcher    *Watcher
	localPath  string
	bucketName string
	remotePath string
	mu         sync.Mutex
	uploading  map[string]struct{}
	OnUpload   func(path string, err error)
}

// NewAutoUploader creates a watcher that automatically uploads changed files
func NewAutoUploader(client *b2.Client, localPath, bucketName, remotePath string, opts *WatcherOptions) (*AutoUploader, error) {
	if opts == nil {
		opts = DefaultWatcherOptions()
	}

	au := &AutoUploader{
		client:     client,
		localPath:  localPath,
		bucketName: bucketName,
		remotePath: remotePath,
		uploading:  make(map[string]struct{}),
	}

	// Set up event handler
	opts.OnEvent = au.handleEvent

	watcher, err := NewWatcher(opts)
	if err != nil {
		return nil, err
	}

	au.watcher = watcher
	return au, nil
}

// Start begins watching and uploading
func (au *AutoUploader) Start(ctx context.Context) error {
	return au.watcher.Watch(ctx, au.localPath)
}

// Stop stops the auto uploader
func (au *AutoUploader) Stop() {
	au.watcher.Stop()
}

// handleEvent handles file system events by uploading files
func (au *AutoUploader) handleEvent(event Event) {
	// Only handle create and write events
	if event.Op != Create && event.Op != Write {
		return
	}

	// Check if file exists and is not a directory
	info, err := os.Stat(event.Path)
	if err != nil || info.IsDir() {
		return
	}

	// Prevent concurrent uploads of the same file
	au.mu.Lock()
	if _, uploading := au.uploading[event.Path]; uploading {
		au.mu.Unlock()
		return
	}
	au.uploading[event.Path] = struct{}{}
	au.mu.Unlock()

	// Upload in goroutine
	go func() {
		defer func() {
			au.mu.Lock()
			delete(au.uploading, event.Path)
			au.mu.Unlock()
		}()

		// Calculate remote path
		relPath, err := filepath.Rel(au.localPath, event.Path)
		if err != nil {
			if au.OnUpload != nil {
				au.OnUpload(event.Path, err)
			}
			return
		}

		remotePath := filepath.ToSlash(filepath.Join(au.remotePath, relPath))

		// Open file
		f, err := os.Open(event.Path)
		if err != nil {
			if au.OnUpload != nil {
				au.OnUpload(event.Path, err)
			}
			return
		}
		defer f.Close()

		// Upload
		err = au.client.Upload(context.Background(), au.bucketName, remotePath, f, info.Size(), nil)
		if au.OnUpload != nil {
			au.OnUpload(event.Path, err)
		}
	}()
}
