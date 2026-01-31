package watch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// ChangeEvent represents a filesystem change.
type ChangeEvent struct {
	Path       string
	ChangeType string // "create", "write", "remove", "rename"
}

// FSWatcher watches a directory tree for filesystem changes using fsnotify.
type FSWatcher struct {
	watcher  *fsnotify.Watcher
	debounce time.Duration
	onChange func(ChangeEvent)
}

// NewFSWatcher creates a new filesystem watcher.
func NewFSWatcher(debounce time.Duration, onChange func(ChangeEvent)) (*FSWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create fsnotify watcher: %w", err)
	}
	if debounce == 0 {
		debounce = 500 * time.Millisecond
	}
	return &FSWatcher{
		watcher:  w,
		debounce: debounce,
		onChange: onChange,
	}, nil
}

// WatchRecursive adds a directory and all its subdirectories to the watcher.
func (w *FSWatcher) WatchRecursive(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if err := w.watcher.Add(path); err != nil {
				return fmt.Errorf("watch %s: %w", path, err)
			}
		}
		return nil
	})
}

// Run starts the event loop. It blocks until the context is cancelled.
func (w *FSWatcher) Run(ctx context.Context) error {
	defer w.watcher.Close()

	var lastEvent ChangeEvent
	debouncer := NewDebouncer(w.debounce, func() {
		if w.onChange != nil {
			w.onChange(lastEvent)
		}
	})
	defer debouncer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-w.watcher.Events:
			if !ok {
				return nil
			}
			changeType := opToChangeType(event.Op)
			if changeType == "" {
				continue
			}

			// If a new directory was created, watch it recursively
			if event.Op.Has(fsnotify.Create) {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					_ = w.WatchRecursive(event.Name)
				}
			}

			lastEvent = ChangeEvent{Path: event.Name, ChangeType: changeType}
			debouncer.Trigger()

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return nil
			}
			return fmt.Errorf("watcher error: %w", err)
		}
	}
}

func opToChangeType(op fsnotify.Op) string {
	switch {
	case op.Has(fsnotify.Create):
		return "create"
	case op.Has(fsnotify.Write):
		return "write"
	case op.Has(fsnotify.Remove):
		return "remove"
	case op.Has(fsnotify.Rename):
		return "rename"
	default:
		return ""
	}
}
