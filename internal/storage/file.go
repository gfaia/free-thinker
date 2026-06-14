package storage

import (
	"fmt"
	"os"
	"path/filepath"
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
