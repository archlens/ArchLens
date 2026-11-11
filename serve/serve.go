// Package serve runs a local HTTP server that shows the diagram in the browser
// and reloads it whenever files in the project change.
package serve

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Build re-renders the project and returns the finished HTML page
// (use html.Exporter{Live: true} so the page listens for reload events).
type Build func() ([]byte, error)

const debounce = 250 * time.Millisecond

type server struct {
	mu   sync.RWMutex
	page []byte
	subs map[chan struct{}]struct{}
}

func newServer(page []byte) *server {
	return &server{page: page, subs: map[chan struct{}]struct{}{}}
}

// update swaps in a new page and tells every connected browser to reload.
func (s *server) update(page []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.page = page
	for c := range s.subs {
		select {
		case c <- struct{}{}:
		default: // a reload is already pending for this client
		}
	}
}

func (s *server) handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		s.mu.RLock()
		page := s.page
		s.mu.RUnlock()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(page)
	})

	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		fl, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-store")

		c := make(chan struct{}, 1)
		s.mu.Lock()
		s.subs[c] = struct{}{}
		s.mu.Unlock()
		defer func() {
			s.mu.Lock()
			delete(s.subs, c)
			s.mu.Unlock()
		}()

		fl.Flush() // send headers so EventSource marks the connection open
		for {
			select {
			case <-c:
				fmt.Fprint(w, "event: reload\ndata: 1\n\n")
				fl.Flush()
			case <-r.Context().Done():
				return
			}
		}
	})

	return mux
}

// Run serves the diagram on addr and rebuilds it whenever something under root
// changes. initial is the first rendered page. ignore holds file or directory
// base names to skip (e.g. ".git", ".archlens", "diagrams"). It blocks until
// ctx is cancelled or the server fails.
func Run(ctx context.Context, addr, root string, ignore map[string]bool, build Build, initial []byte) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()

	// fsnotify is not recursive, so add every directory ourselves.
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if p != root && ignore[d.Name()] {
			return filepath.SkipDir
		}
		return w.Add(p)
	})
	if err != nil {
		return err
	}

	srv := newServer(initial)
	hs := &http.Server{Addr: addr, Handler: srv.handler()}

	go watchLoop(ctx, w, ignore, build, srv)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = hs.Shutdown(shutdownCtx)
	}()

	log.Printf("watching %s, serving http://%s", root, addr)
	if err := hs.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func watchLoop(ctx context.Context, w *fsnotify.Watcher, ignore map[string]bool, build Build, srv *server) {
	timer := time.NewTimer(time.Hour)
	timer.Stop()

	for {
		select {
		case ev, ok := <-w.Events:
			if !ok {
				return
			}
			if ev.Has(fsnotify.Chmod) || ignore[filepath.Base(ev.Name)] {
				continue
			}
			// Start watching newly created directories.
			if ev.Has(fsnotify.Create) {
				if fi, err := os.Stat(ev.Name); err == nil && fi.IsDir() {
					_ = w.Add(ev.Name)
				}
			}
			timer.Reset(debounce) // editors emit bursts of events per save

		case <-timer.C:
			page, err := build()
			if err != nil {
				log.Println("build failed, keeping previous diagram:", err)
				continue
			}
			srv.update(page)

		case err, ok := <-w.Errors:
			if !ok {
				return
			}
			log.Println("watch error:", err)

		case <-ctx.Done():
			return
		}
	}
}
