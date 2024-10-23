package fsys

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// MemoryStorage is an in-memory implementation of the FS interface.
type MemoryStorage struct {
	files map[string]*File
	mu    sync.RWMutex
}

// File represents an in-memory file.
type File struct {
	Name    string
	Content *bytes.Buffer
}

// NewMemoryStorage returns a new MemoryStorage instance.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		files: make(map[string]*File),
	}
}

// Driver returns the name of the current driver.
func (fs *MemoryStorage) Driver() string {
	return DRIVER_MEMORY
}

// Read reads a file from memory.
func (fs *MemoryStorage) Read(path string) (io.ReadCloser, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	file, exists := fs.files[path]
	if !exists {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(bytes.NewReader(file.Content.Bytes())), nil
}

// Write writes a file to memory.
func (fs *MemoryStorage) Write(path string, contents []byte) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.files[path] = &File{
		Name:    path,
		Content: bytes.NewBuffer(contents),
	}
	return nil
}

// Delete deletes a file from memory.
func (fs *MemoryStorage) Delete(path string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if _, exists := fs.files[path]; !exists {
		return os.ErrNotExist
	}
	delete(fs.files, path)
	return nil
}

// Exists checks if a file exists in memory.
func (fs *MemoryStorage) Exists(path string) (bool, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	_, exists := fs.files[path]
	return exists, nil
}

// Rename renames a file in memory.
func (fs *MemoryStorage) Rename(oldPath, newPath string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	file, exists := fs.files[oldPath]
	if !exists {
		return os.ErrNotExist
	}

	// Perform the rename
	fs.files[newPath] = file
	delete(fs.files, oldPath)
	return nil
}

// Copy copies a file in memory.
func (fs *MemoryStorage) Copy(sourcePath, destinationPath string) error {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	sourceFile, exists := fs.files[sourcePath]
	if !exists {
		return os.ErrNotExist
	}

	// Copy content
	fs.files[destinationPath] = &File{
		Name:    destinationPath,
		Content: bytes.NewBuffer(sourceFile.Content.Bytes()),
	}
	return nil
}

// CreateDirectory is a no-op for memory storage but can simulate directory creation.
func (fs *MemoryStorage) CreateDirectory(path string) error {
	// Since this is in-memory storage, directory creation can be simulated as a prefix check.
	if !strings.HasSuffix(path, "/") {
		return errors.New("directory path must end with '/'")
	}
	return nil
}

// GetUrl returns a mock URL for memory-stored files.
func (fs *MemoryStorage) GetUrl(path string) (string, error) {
	// In-memory storage doesn't have real URLs, so return a mock URL.
	if exists, _ := fs.Exists(path); !exists {
		return "", os.ErrNotExist
	}
	return "mem://" + path, nil
}

// Open opens a file (not fully applicable for memory but returns a mock).
func (fs *MemoryStorage) Open(path string) (*os.File, error) {
	return nil, errors.New("open is not supported for in-memory storage")
}

// Upload simulates file upload by storing the uploaded file's content.
func (fs *MemoryStorage) Upload(file multipart.File, header *multipart.FileHeader, dir string) (*os.File, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	var buf bytes.Buffer
	_, err := io.Copy(&buf, file)
	if err != nil {
		return nil, err
	}

	// Simulate storing the uploaded file in memory
	filePath := filepath.Join(dir, header.Filename)
	fs.files[filePath] = &File{
		Name:    filePath,
		Content: &buf,
	}

	// Since this is in-memory, there's no real os.File to return.
	return nil, nil
}
