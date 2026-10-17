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

// The exact production failure, as a test.
//
// A fresh deployment has never written an upload, so the disk root —
// storage/app/public — does not exist. The old guard walked up from the file
// looking for something that did exist, found storage (made by the logger) or
// the working directory, and reported "resolves outside local storage root":
// a traversal error for a path that traverses nothing. The real answer was
// that the directory had never been created, and the message pointed at the
// wrong thing entirely.
func TestWritingToADiskWhoseRootDoesNotExistYet(t *testing.T) {
	base := t.TempDir()
	// Nested two deep and never created, which is the shape productshore
	// configures: STORAGE_PATH/app/public.
	root := filepath.Join(base, "app", "public")

	ls := NewLocalStorage(root)

	if err := ls.Write("avatars/1-1792230349508_sq.jpg", []byte("jpeg")); err != nil {
		t.Fatalf("writing to an uncreated root: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(root, "avatars", "1-1792230349508_sq.jpg"))
	if err != nil {
		t.Fatalf("reading it back: %v", err)
	}
	if string(got) != "jpeg" {
		t.Errorf("contents = %q, want jpeg", got)
	}
}

// And the guard still refuses what it exists to refuse. Stopping the walk at
// the root narrows where it looks, not what it rejects.
func TestTraversalIsStillRefusedWhenTheRootIsMissing(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "app", "public")
	ls := NewLocalStorage(root)

	for _, path := range []string{
		"../escaped.txt",
		"avatars/../../escaped.txt",
		"a/b/../../../escaped.txt",
	} {
		if err := ls.Write(path, []byte("x")); err == nil {
			t.Errorf("Write(%q) was allowed out of the root", path)
		}
	}
	if _, err := os.Stat(filepath.Join(base, "escaped.txt")); err == nil {
		t.Error("a write escaped the root")
	}
}

// A symlink out of the root is the case the walk is actually for, and it has
// to keep working once the root exists.
func TestASymlinkOutOfTheRootIsRefused(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "root")
	outside := filepath.Join(base, "outside")
	for _, dir := range []string{root, outside} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	ls := NewLocalStorage(root)
	if err := ls.Write("escape/leaked.txt", []byte("x")); err == nil {
		t.Error("a write followed a symlink out of the root")
	}
	if _, err := os.Stat(filepath.Join(outside, "leaked.txt")); err == nil {
		t.Error("the file landed outside the root")
	}
}
