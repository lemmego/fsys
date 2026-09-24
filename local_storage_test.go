package fsys

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLocalStorageResolvePath(t *testing.T) {
	root := t.TempDir()
	storage := NewLocalStorage(root)

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "normal path", path: "documents/report.txt", want: true},
		{name: "normalized path within root", path: "documents/../report.txt", want: true},
		{name: "parent traversal", path: "../outside.txt", want: false},
		{name: "nested parent traversal", path: "documents/../../outside.txt", want: false},
		{name: "absolute path", path: filepath.Join(root, "outside.txt"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := storage.resolvePath(tt.path)
			if (err == nil) != tt.want {
				t.Fatalf("resolvePath(%q) error = %v, want allowed = %v", tt.path, err, tt.want)
			}
		})
	}
}

func TestLocalStorageRejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on some Windows configurations")
	}

	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}

	storage := NewLocalStorage(root)
	if _, err := storage.Read("linked/secret.txt"); err == nil {
		t.Fatal("Read through an escaping symlink succeeded")
	}
	if err := storage.Write("linked/new.txt", []byte("not allowed")); err == nil {
		t.Fatal("Write through an escaping symlink succeeded")
	}
}

func TestLocalStorageAllowsNormalOperations(t *testing.T) {
	storage := NewLocalStorage(t.TempDir())
	if err := storage.CreateDirectory("documents"); err != nil {
		t.Fatal(err)
	}
	if err := storage.Write("documents/report.txt", []byte("report")); err != nil {
		t.Fatal(err)
	}

	exists, err := storage.Exists("documents/report.txt")
	if err != nil || !exists {
		t.Fatalf("Exists() = %t, %v, want true, nil", exists, err)
	}

	contents, err := os.ReadFile(filepath.Join(storage.RootDirectory, "documents/report.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "report" {
		t.Fatalf("stored contents = %q, want %q", contents, "report")
	}
}
