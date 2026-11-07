package config

import (
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches config files and calls reload after debounce.
type Watcher struct {
	watcher  *fsnotify.Watcher
	dir      string
	debounce time.Duration
	reload   func()
	stop     chan struct{}
}

// NewWatcher creates a watcher for dir.
func NewWatcher(dir string, debounce time.Duration, reload func()) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := w.Add(dir); err != nil {
		_ = w.Close()
		return nil, err
	}
	if debounce <= 0 {
		debounce = 200 * time.Millisecond
	}
	return &Watcher{watcher: w, dir: dir, debounce: debounce, reload: reload, stop: make(chan struct{})}, nil
}

// Start begins watching in a goroutine.
func (w *Watcher) Start() {
	go func() {
		var timer *time.Timer
		for {
			select {
			case <-w.stop:
				if timer != nil {
					timer.Stop()
				}
				return
			case ev, ok := <-w.watcher.Events:
				if !ok {
					return
				}
				if ev.Op&(fsnotify.Write|fsnotify.Create) == 0 {
					continue
				}
				if filepath.Base(ev.Name) == "" {
					continue
				}
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(w.debounce, func() {
					if w.reload != nil {
						w.reload()
					}
				})
			case _, ok := <-w.watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()
}

// Stop stops the watcher.
func (w *Watcher) Stop() error {
	close(w.stop)
	return w.watcher.Close()
}
