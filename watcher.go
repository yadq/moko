package main

import (
	"github.com/fsnotify/fsnotify"
	"github.com/gookit/slog"
)

type FileWatcher struct {
	w              *fsnotify.Watcher
	m              map[string]func() error // watcher map
	watchingEvents []fsnotify.Op
	stopEvt        chan struct{}
	stoppedEvt     chan struct{}
}

func NewFileWatcher() *FileWatcher {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Fatalf("new watcher error: %v", err)
	}
	nw := &FileWatcher{
		w: w,
		m: map[string]func() error{},
		watchingEvents: []fsnotify.Op{
			fsnotify.Create,
			fsnotify.Rename,
			fsnotify.Remove,
			fsnotify.Write,
		},
		stopEvt:    make(chan struct{}),
		stoppedEvt: make(chan struct{}),
	}

	go nw.watching()

	return nw
}

func (nw *FileWatcher) Remove(filepath string) error {
	return nw.w.Remove(filepath)
}

// On add handler
func (nw *FileWatcher) Watch(file string, handler func() error) {
	slog.Infof("add watcher %s", file)
	if err := nw.w.Add(file); err != nil {
		slog.Errorf("add watcher error: %v", err)
		return
	}
	nw.m[file] = handler
}

// Stop stop the watching goroutine
func (nw *FileWatcher) Stop() {
	close(nw.stopEvt)
	<-nw.stoppedEvt
	slog.Info("watcher stopped")
}

func (nw *FileWatcher) watching() {
	defer func() {
		close(nw.stoppedEvt)
	}()
	defer nw.w.Close()

	for {
		select {
		case event, ok := <-nw.w.Events:
			if !ok {
				slog.Error("failed to read watch event")
				return
			}
			slog.Infof("got event: %v", event)
			handler, exists := nw.m[event.Name]
			if !exists {
				continue
			}
			for _, evt := range nw.watchingEvents {
				if event.Op&evt == evt {
					// if evt != fsnotify.Write { // create | rename | remove
					// 	slog.Infof("add new watcher %s", event.Name)
					// 	if err := nw.w.Add(event.Name); err != nil {
					// 		slog.Errorf("add new watcher error: %v", err)
					// 		continue
					// 	}
					// }
					if err := handler(); err != nil {
						slog.Errorf("handle watcher %s error, %v", event.Name, err)
					}
				}
			}
		case err, ok := <-nw.w.Errors:
			if !ok {
				return
			}
			slog.Errorf("got watcher error %v", err)
		case <-nw.stopEvt:
			slog.Warn("got stop watcher event")
			return
		}
	}
}
