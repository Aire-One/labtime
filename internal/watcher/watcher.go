package watcher

import (
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
)

type update int

type Watcher struct {
	watcher *fsnotify.Watcher

	Events chan update
	Errors chan error
}

func NewWatcher(file string) (*Watcher, error) {
	if err := validateFile(file); err != nil {
		return nil, errors.Wrapf(err, "error validating file %q", file)
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, errors.Wrapf(err, "error creating fsnotify watcher")
	}

	eventchan := make(chan update, 1)
	errchan := make(chan error, 1)
	go fileLoop(w, file, eventchan, errchan)

	// Watch the directory, not the file itself.
	err = w.Add(filepath.Dir(file))
	if err != nil {
		return nil, errors.Wrapf(err, "error adding file %q to watcher", file)
	}

	return &Watcher{
		watcher: w,
		Events:  eventchan,
		Errors:  errchan,
	}, nil
}

func validateFile(file string) error {
	st, err := os.Lstat(file)
	if err != nil {
		return errors.Wrapf(err, "error getting file info for %q", file)
	}

	if st.IsDir() {
		return errors.Errorf("expected a file, but %q is a directory", file)
	}

	return nil
}

func fileLoop(w *fsnotify.Watcher, file string, eventchan chan update, errchan chan error) {
	for {
		select {
		case err, ok := <-w.Errors:
			if !ok { // Channel was closed (i.e. Watcher.Close() was called).
				return
			}
			errchan <- errors.Wrapf(err, "error received from fsnotify watcher")
		case e, ok := <-w.Events:
			if !ok { // Channel was closed (i.e. Watcher.Close() was called).
				return
			}

			if e.Name != file || e.Op != fsnotify.Write {
				continue
			}

			eventchan <- 1
		}
	}
}

func (w *Watcher) Shutdown() error {
	if err := w.watcher.Close(); err != nil {
		return errors.Wrap(err, "error closing fsnotify watcher")
	}

	return nil
}
