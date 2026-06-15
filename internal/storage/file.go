package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FileStore struct {
	root string
}

func NewFileStore(root string) *FileStore {
	return &FileStore{root: root}
}

func (fs *FileStore) Root() string { return fs.root }

func (fs *FileStore) Save(source, content string) (string, error) {
	if content == "" {
		return "", nil
	}
	date := time.Now().UTC().Format("2006-01-02")
	dir := filepath.Join(fs.root, source, date)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", dir, err)
	}

	filename := fmt.Sprintf("%d.html", time.Now().UnixNano())
	full := filepath.Join(dir, filename)
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}
	return full, nil
}

func (fs *FileStore) Read(path string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("empty content path")
	}

	root, err := filepath.Abs(fs.root)
	if err != nil {
		return nil, fmt.Errorf("resolve storage root: %w", err)
	}
	target := filepath.Clean(path)
	if !filepath.IsAbs(target) {
		absTarget, err := filepath.Abs(target)
		if err != nil {
			return nil, fmt.Errorf("resolve content path: %w", err)
		}
		if isWithin(root, absTarget) {
			target = absTarget
		} else {
			target = filepath.Join(root, target)
		}
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return nil, fmt.Errorf("resolve content path: %w", err)
	}

	if !isWithin(root, target) {
		return nil, fmt.Errorf("content path outside storage root")
	}

	b, err := os.ReadFile(target)
	if err != nil {
		return nil, fmt.Errorf("read content file: %w", err)
	}
	return b, nil
}

func isWithin(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}
