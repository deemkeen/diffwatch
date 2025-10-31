package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Event represents a file system change event
type Event struct {
	Path      string
	Op        string // "create", "write", "remove", "rename", "chmod"
	Timestamp time.Time
}

// FileWatcher watches files for changes and emits debounced events
type FileWatcher struct {
	watcher   *fsnotify.Watcher
	events    chan Event
	errors    chan error
	debouncer *Debouncer
	mu        sync.RWMutex
	closed    bool
	recursive bool
	watchPath string
}

// New creates a new FileWatcher for the given path
func New(path string, recursive bool) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating watcher: %w", err)
	}

	// Add the path to watch
	absPath, err := filepath.Abs(path)
	if err != nil {
		watcher.Close()
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	fw := &FileWatcher{
		watcher:   watcher,
		events:    make(chan Event, 100),
		errors:    make(chan error, 10),
		debouncer: NewDebouncer(100 * time.Millisecond),
		recursive: recursive,
		watchPath: absPath,
	}

	// Add paths to watch
	if recursive {
		if err := fw.addRecursive(absPath); err != nil {
			watcher.Close()
			return nil, err
		}
	} else {
		if err := watcher.Add(absPath); err != nil {
			watcher.Close()
			return nil, fmt.Errorf("adding path to watcher: %w", err)
		}
	}

	// Start watching in background
	go fw.watch()

	return fw, nil
}

// addRecursive adds a directory and all its subdirectories to the watcher
func (fw *FileWatcher) addRecursive(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip directories/files with permission errors
			if os.IsPermission(err) {
				return filepath.SkipDir
			}
			return err
		}
		if info.IsDir() {
			if err := fw.watcher.Add(path); err != nil {
				// Skip if permission denied
				if os.IsPermission(err) {
					return filepath.SkipDir
				}
				return fmt.Errorf("adding path to watcher: %w", err)
			}
		}
		return nil
	})
}

// Events returns the channel of debounced file events
func (fw *FileWatcher) Events() <-chan Event {
	return fw.events
}

// Errors returns the channel of errors
func (fw *FileWatcher) Errors() <-chan error {
	return fw.errors
}

// Close stops the watcher and releases resources
func (fw *FileWatcher) Close() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.closed {
		return nil
	}

	fw.closed = true
	fw.debouncer.Stop()
	close(fw.events)
	close(fw.errors)
	return fw.watcher.Close()
}

// watch runs in a goroutine and processes file system events
func (fw *FileWatcher) watch() {
	for {
		select {
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}
			fw.handleEvent(event)

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			fw.sendError(err)
		}
	}
}

// handleEvent processes a raw fsnotify event
func (fw *FileWatcher) handleEvent(event fsnotify.Event) {
	op := opToString(event.Op)

	// If recursive mode and a directory was created, add it to the watcher
	if fw.recursive && event.Op&fsnotify.Create == fsnotify.Create {
		if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
			if err := fw.addRecursive(event.Name); err != nil {
				// Only send error if it's not a permission error
				if !os.IsPermission(err) {
					fw.sendError(fmt.Errorf("adding new directory to watcher: %w", err))
				}
			}
		}
	}

	ev := Event{
		Path:      event.Name,
		Op:        op,
		Timestamp: time.Now(),
	}

	// Debounce the event
	fw.debouncer.Add(event.Name, func() {
		fw.sendEvent(ev)
	})
}

// sendEvent safely sends an event to the events channel
func (fw *FileWatcher) sendEvent(event Event) {
	fw.mu.RLock()
	defer fw.mu.RUnlock()

	if fw.closed {
		return
	}

	select {
	case fw.events <- event:
	default:
		// Channel full, drop event (backpressure)
	}
}

// sendError safely sends an error to the errors channel
func (fw *FileWatcher) sendError(err error) {
	fw.mu.RLock()
	defer fw.mu.RUnlock()

	if fw.closed {
		return
	}

	select {
	case fw.errors <- err:
	default:
		// Channel full, drop error
	}
}

// opToString converts fsnotify.Op to a string
func opToString(op fsnotify.Op) string {
	switch {
	case op&fsnotify.Create == fsnotify.Create:
		return "create"
	case op&fsnotify.Write == fsnotify.Write:
		return "write"
	case op&fsnotify.Remove == fsnotify.Remove:
		return "remove"
	case op&fsnotify.Rename == fsnotify.Rename:
		return "rename"
	case op&fsnotify.Chmod == fsnotify.Chmod:
		return "chmod"
	default:
		return "unknown"
	}
}
