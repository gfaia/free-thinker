package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileStoreRead(t *testing.T) {
	fs := NewFileStore(t.TempDir())
	path, err := fs.Save("zhihu", "<html>hello</html>")
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	b, err := fs.Read(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(b) != "<html>hello</html>" {
		t.Fatalf("content=%q", string(b))
	}
}

func TestFileStoreReadRejectsUnsafePaths(t *testing.T) {
	root := t.TempDir()
	fs := NewFileStore(root)

	if _, err := fs.Read(""); err == nil {
		t.Fatal("expected empty path error")
	}

	outside := filepath.Join(t.TempDir(), "outside.html")
	if err := os.WriteFile(outside, []byte("secret"), 0o644); err != nil {
		t.Fatalf("write outside: %v", err)
	}
	_, err := fs.Read(outside)
	if err == nil || !strings.Contains(err.Error(), "outside storage root") {
		t.Fatalf("err=%v, want outside storage root", err)
	}
}

func TestFileStoreReadMissingFile(t *testing.T) {
	fs := NewFileStore(t.TempDir())
	if _, err := fs.Read("missing.html"); err == nil {
		t.Fatal("expected missing file error")
	}
}
