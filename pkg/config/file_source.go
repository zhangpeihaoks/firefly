package config

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

func init() {
	RegisterSource("file", newFileSourceFromURL)
}

// FileSource is a RemoteSource that reads a local YAML file.
//
// The first Watch call returns the current content immediately; subsequent
// calls block until the file changes (watching the parent directory to survive
// atomic replace), then return the updated configuration.
type FileSource struct {
	path    string
	started atomic.Bool
}

// NewFileSource creates a file configuration source.
func NewFileSource(path string) *FileSource {
	return &FileSource{path: path}
}

// Name returns the source name.
func (s *FileSource) Name() string {
	return "file:" + s.path
}

// Watch returns the current file content immediately on the first call, then
// blocks until the file changes on subsequent calls.
func (s *FileSource) Watch(ctx context.Context) (map[string]any, error) {
	// First call: return the current configuration immediately.
	if !s.started.CompareAndSwap(false, true) {
		return s.watchChanges(ctx)
	}
	return readYAMLFile(s.path)
}

// watchChanges blocks until the watched file changes, then returns the new content.
func (s *FileSource) watchChanges(ctx context.Context) (map[string]any, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("file source: create watcher: %w", err)
	}
	defer watcher.Close()

	// Watch the parent directory: files may be atomically replaced
	// (write temp + rename), which breaks watching the file itself.
	dir := filepath.Dir(s.path)
	if err := watcher.Add(dir); err != nil {
		return nil, fmt.Errorf("file source: watch directory %q: %w", dir, err)
	}

	base := filepath.Base(s.path)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case event, ok := <-watcher.Events:
			if !ok {
				return nil, fmt.Errorf("file source: watcher closed")
			}
			if filepath.Base(event.Name) != base {
				continue
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) == 0 {
				continue
			}
			// Debounce: the file may be mid-replace; retry a few times.
			time.Sleep(50 * time.Millisecond)
			for i := 0; i < 3; i++ {
				if data, err := readYAMLFile(s.path); err == nil {
					return data, nil
				}
				time.Sleep(50 * time.Millisecond)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil, fmt.Errorf("file source: watcher closed")
			}
			return nil, fmt.Errorf("file source: watch error: %w", err)
		}
	}
}

// readYAMLFile reads a YAML file into a nested map.
func readYAMLFile(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("file source: read %q: %w", path, err)
	}
	var m map[string]any
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("file source: parse %q: %w", path, err)
	}
	return m, nil
}

// newFileSourceFromURL builds a FileSource from a file:// URL.
// Windows paths like file:///D:/app/config.yaml are normalized.
func newFileSourceFromURL(_ context.Context, u *url.URL) (RemoteSource, error) {
	path := u.Path
	if path == "" {
		// file://host/path form
		path = u.Host + u.Path
	}
	if runtime.GOOS == "windows" {
		path = strings.TrimPrefix(path, "/")
	}
	if path == "" {
		return nil, fmt.Errorf("file source: empty path in URL %q", u.String())
	}
	return NewFileSource(path), nil
}
