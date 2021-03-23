package watcher

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Listener is an interface for receivers of filesystem events.
type Listener interface {
	// Refresh is invoked by the watcher on relevant filesystem events.
	// If the listener returns an error, it is served to the user on the current request.
	Refresh() error
}

// Watcher allows listeners to register to be notified of changes under a given
// directory.
type Watcher struct {
	// Parallel arrays of watcher/listener pairs.
	watchers    []*fsnotify.Watcher
	listeners   []Listener
	lastError   int
	notifyMutex sync.Mutex
}

func New() *Watcher {
	return &Watcher{
		// forceRefresh: true,
		lastError: -1,
	}
}

// Listen registers for events within the given root directories (recursively).
func (w *Watcher) Listen(listener Listener, roots ...string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	// Replace the unbuffered Event channel with a buffered one.
	// Otherwise multiple change events only come out one at a time, across
	// multiple page views.  (There appears no way to "pump" the events out of
	// the watcher)
	watcher.Events = make(chan fsnotify.Event, 100)
	watcher.Errors = make(chan error, 10)

	// Walk through all files / directories under the root, adding each to watcher.
	for _, p := range roots {
		// is the directory / file a symlink?
		f, err := os.Lstat(p)
		if err == nil && f.Mode()&os.ModeSymlink == os.ModeSymlink {
			realPath, err := filepath.EvalSymlinks(p)
			if err != nil {
				panic(err)
			}
			p = realPath
		}

		fi, err := os.Stat(p)
		if err != nil {
			return fmt.Errorf("Failed to stat watched path %s: %w", p, err)
		}

		// If it is a file, watch that specific file.
		if !fi.IsDir() {
			err = watcher.Add(p)
			if err != nil {
				return fmt.Errorf("Failed to watch %s: %w", p, err)
			}
			continue
		}

		var watcherWalker func(path string, info os.FileInfo, err error) error

		watcherWalker = func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// is it a symlinked template?
			link, err := os.Lstat(path)
			if err == nil && link.Mode()&os.ModeSymlink == os.ModeSymlink {
				// lookup the actual target & check for goodness
				targetPath, err := filepath.EvalSymlinks(path)
				if err != nil {
					return fmt.Errorf("failed to read symlink %s: %w", path, err)
				}
				targetInfo, err := os.Stat(targetPath)
				if err != nil {
					return fmt.Errorf("failed to stat symlink target %s of %s: %w", targetPath, path, err)
				}

				// set the template path to the target of the symlink
				path = targetPath
				info = targetInfo
				if err := filepath.Walk(path, watcherWalker); err != nil {
					return err
				}
			}

			if info.IsDir() {
				if err := watcher.Add(path); err != nil {
					return err
				}
			}
			return nil
		}

		// Else, walk the directory tree.
		if err := filepath.Walk(p, watcherWalker); err != nil {
			return fmt.Errorf("error walking path %s: %w", p, err)
		}
	}

	w.watchers = append(w.watchers, watcher)
	w.listeners = append(w.listeners, listener)

	return nil
}

// Notify causes the watcher to forward any change events to listeners.
// It returns the first (if any) error returned.
func (w *Watcher) Notify() error {
	// Serialize Notify() calls.
	w.notifyMutex.Lock()
	defer w.notifyMutex.Unlock()

	for idx, watcher := range w.watchers {
		listener := w.listeners[idx]

		// Pull all pending events / errors from the watcher.
		refresh := false
		for {
			select {
			case ev := <-watcher.Events:
				// Ignore changes to dotfiles.
				if !strings.HasPrefix(path.Base(ev.Name), ".") {
					refresh = true
				}
				continue
			case <-watcher.Errors:
				continue
			default:
				// No events left to pull
			}
			break
		}

		if refresh || w.lastError == idx {
			err := listener.Refresh()
			if err != nil {
				w.lastError = idx
				return err
			}
		}
	}

	w.lastError = -1
	return nil
}
