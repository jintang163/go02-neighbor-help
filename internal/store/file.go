package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type FileStore struct {
	mem  *MemoryStore
	path string
	saveMu sync.Mutex
}

func NewFileStore(path string) (*FileStore, error) {
	mem := NewMemoryStore(nil, nil)
	fs := &FileStore{mem: mem, path: path}
	mem.SetPersistHook(fs.save)
	if err := fs.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load store %q: %w", path, err)
	}
	return fs, nil
}

func (fs *FileStore) Store() *MemoryStore { return fs.mem }

func (fs *FileStore) Path() string { return fs.path }

func (fs *FileStore) load() error {
	data, err := os.ReadFile(fs.path)
	if err != nil {
		return err
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return fmt.Errorf("unmarshal snapshot: %w", err)
	}
	fs.mem.ReplaceAll(snap)
	return nil
}

func (fs *FileStore) save() {
	fs.saveMu.Lock()
	defer fs.saveMu.Unlock()
	snap := fs.mem.Snapshot()
	if err := fs.writeAtomic(snap); err != nil {
		fmt.Fprintf(os.Stderr, "store: persist failed: %v\n", err)
	}
}

func (fs *FileStore) writeAtomic(snap Snapshot) error {
	if dir := filepath.Dir(fs.path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("mkdir: %w", err)
		}
	}
	tmp, err := os.CreateTemp(filepath.Dir(fs.path), ".store-*.json.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }

	enc := json.NewEncoder(tmp)
	if err := enc.Encode(snap); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("encode: %w", err)
	}
	if err := tmp.Sync(); err != nil && !errors.Is(err, io.ErrNoProgress) {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("sync: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpName, fs.path); err != nil {
		cleanup()
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

func (fs *FileStore) Flush() error {
	fs.saveMu.Lock()
	defer fs.saveMu.Unlock()
	return fs.writeAtomic(fs.mem.Snapshot())
}

var _ = context.Background
