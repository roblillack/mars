package watcher

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type SimpleRefresher struct {
	Refreshed bool
	Error     error
}

func (l *SimpleRefresher) Refresh() error {
	l.Refreshed = true
	return l.Error
}

func TestWatcher(t *testing.T) {
	w := New()

	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("mars-watcher-test-%d", rand.Uint32()))
	err := os.MkdirAll(tmp, 0700)
	if err != nil {
		t.Fatal(err)
	}

	bla := &SimpleRefresher{}
	if err := w.Listen(bla, tmp); err != nil {
		t.Errorf("unable to setup listener: %s", err)
	}

	if err := w.Notify(); err != nil {
		t.Errorf("unable to notify listeners: %s", err)
	}
	if bla.Refreshed {
		t.Error("No changes to tmp dir yet, should not have been refreshed.")
	}

	bla.Refreshed = false
	if f, err := os.Create(filepath.Join(tmp, "yep.dada")); err != nil {
		t.Fatal(err)
	} else {
		fmt.Fprintln(f, "Hello world!")
		f.Close()
	}

	time.Sleep(1 * time.Second)

	if err := w.Notify(); err != nil {
		t.Errorf("unable to notify listeners: %s", err)
	}
	if !bla.Refreshed {
		t.Error("Should have been refreshed.")
	}

	if err := os.RemoveAll(tmp); err != nil {
		t.Fatal(err)
	}
}

func TestErrorWhileRefreshing(t *testing.T) {
	w := New()

	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("mars-watcher-test-%d", rand.Uint32()))
	err := os.MkdirAll(tmp, 0700)
	if err != nil {
		t.Fatal(err)
	}

	bla := &SimpleRefresher{Error: errors.New("uh-oh something went wrong!!!11")}
	if err := w.Listen(bla, tmp); err != nil {
		t.Errorf("unable to setup listener: %s", err)
	}

	if err := w.Notify(); err != nil {
		t.Errorf("unable to notify listeners: %s", err)
	}
	if bla.Refreshed {
		t.Error("No changes to tmp dir yet, should not have been refreshed.")
	}

	bla.Refreshed = false
	if f, err := os.Create(filepath.Join(tmp, "yep.dada")); err != nil {
		t.Fatal(err)
	} else {
		fmt.Fprintln(f, "Hello world!")
		f.Close()
	}

	time.Sleep(1 * time.Second)

	if err := w.Notify(); err == nil {
		t.Error("No error while refreshing")
	} else if err != bla.Error {
		t.Error("Wrong error seen while refreshing: %w", err)
	}
	if !bla.Refreshed {
		t.Error("Should have been refreshed.")
	}

	bla.Refreshed = false
	bla.Error = nil
	time.Sleep(1 * time.Second)

	if err := w.Notify(); err != nil {
		t.Errorf("error not resovled yet: %s", err)
	}
	if !bla.Refreshed {
		t.Error("Should have been refreshed.")
	}

	if err := os.RemoveAll(tmp); err != nil {
		t.Fatal(err)
	}
}
