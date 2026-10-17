package fsys

import (
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
)

// LocalStorage is an implementation of StorageInterface for local file system.
type LocalStorage struct {
	// Root directory of the storage.
	RootDirectory string
}

// resolvePath returns a path that is contained by the storage root. It also
// checks symlinks in existing path components so a link cannot escape it.
func (ls *LocalStorage) resolvePath(storagePath string) (string, error) {
	if filepath.IsAbs(storagePath) {
		return "", fmt.Errorf("path %q must be relative to local storage root", storagePath)
	}

	root, err := filepath.Abs(ls.RootDirectory)
	if err != nil {
		return "", err
	}
	root = filepath.Clean(root)
	if resolvedRoot, err := filepath.EvalSymlinks(root); err == nil {
		root = filepath.Clean(resolvedRoot)
	}

	fullPath := filepath.Clean(filepath.Join(root, storagePath))
	if fullPath != root && !hasPathPrefix(fullPath, root) {
		return "", fmt.Errorf("path %q is outside local storage root", storagePath)
	}

	// EvalSymlinks requires the final path to exist. Check the nearest existing
	// parent when this operation is creating a new file or directory.
	//
	// The walk stops at the root and never climbs past it, which is the whole
	// of the fix for a bug this reported as a traversal attempt.
	//
	// A disk whose root does not exist yet is the ordinary state of a fresh
	// deployment: nothing has been written, so storage/app/public is not
	// there. Climbing above the root then finds the first directory that does
	// exist — storage, or the working directory — which is of course not
	// inside the root, and the old code called that "resolves outside local
	// storage root". The message sent you hunting for a path-traversal bug
	// when the real answer was "this directory has never been created", and
	// the two have nothing to do with each other.
	//
	// Stopping at the root is also exactly as strict: every component from
	// the root downwards is still resolved and checked, and the root itself
	// was already resolved above. What is above the root is not this disk's
	// business.
	checkPath := fullPath
	for {
		resolved, evalErr := filepath.EvalSymlinks(checkPath)
		if evalErr == nil {
			if resolved != root && !hasPathPrefix(filepath.Clean(resolved), root) {
				return "", fmt.Errorf("path %q resolves outside local storage root", storagePath)
			}
			break
		}
		if !os.IsNotExist(evalErr) {
			return "", evalErr
		}
		if checkPath == root {
			// The root does not exist. Nothing under it can escape it, and a
			// writer is about to create it.
			break
		}
		parent := filepath.Dir(checkPath)
		if parent == checkPath {
			break
		}
		checkPath = parent
	}

	return fullPath, nil
}

func hasPathPrefix(candidate, root string) bool {
	if root == string(filepath.Separator) {
		return filepath.IsAbs(candidate)
	}
	return len(candidate) > len(root) && candidate[:len(root)] == root && candidate[len(root)] == filepath.Separator
}

func NewLocalStorage(basePath string) *LocalStorage {
	if basePath == "" {
		var err error
		basePath, err = os.Getwd()
		if err != nil {
			panic(err)
		}
	}

	return &LocalStorage{
		RootDirectory: basePath,
	}
}

func (ls *LocalStorage) Driver() string {
	return DRIVER_LOCAL
}

func (ls *LocalStorage) Read(path string) (io.ReadCloser, error) {
	fullPath, err := ls.resolvePath(path)
	if err != nil {
		return nil, err
	}
	return os.Open(fullPath)
}

// Write creates the directories it needs, including the disk root.
//
// os.Create alone fails with ENOENT on a path one directory deep —
// "avatars/x.jpg" on a disk that has never been written to — and every
// caller then has to CreateDirectory first or carry the same MkdirAll.
// Writing a file to a path is the operation being asked for; the directories
// are an implementation detail of it.
//
// resolvePath has already established that fullPath is inside the root, so
// this cannot create a directory anywhere else.
func (ls *LocalStorage) Write(path string, contents []byte) error {
	fullPath, err := ls.resolvePath(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return err
	}
	file, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(contents)
	return err
}

func (ls *LocalStorage) Delete(path string) error {
	fullPath, err := ls.resolvePath(path)
	if err != nil {
		return err
	}
	return os.Remove(fullPath)
}

func (ls *LocalStorage) Exists(path string) (bool, error) {
	fullPath, err := ls.resolvePath(path)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(fullPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func (ls *LocalStorage) Rename(oldPath, newPath string) error {
	oldFullPath, err := ls.resolvePath(oldPath)
	if err != nil {
		return err
	}
	newFullPath, err := ls.resolvePath(newPath)
	if err != nil {
		return err
	}
	return os.Rename(oldFullPath, newFullPath)
}

func (ls *LocalStorage) Copy(sourcePath, destinationPath string) error {
	sourceFullPath, err := ls.resolvePath(sourcePath)
	if err != nil {
		return err
	}
	destinationFullPath, err := ls.resolvePath(destinationPath)
	if err != nil {
		return err
	}
	sourceFile, err := os.Open(sourceFullPath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	if err := os.MkdirAll(filepath.Dir(destinationFullPath), 0o755); err != nil {
		return err
	}
	destinationFile, err := os.Create(destinationFullPath)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	return err
}

func (ls *LocalStorage) CreateDirectory(path string) error {
	// For local storage, use os.MkdirAll, which doesn't return an error if the directory already exists.
	fullPath, err := ls.resolvePath(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		// If the error indicates that the directory already exists, treat it as success
		if os.IsExist(err) {
			return nil
		}
		return err
	}
	return nil
}

func (ls *LocalStorage) GetUrl(path string) (string, error) {
	// Construct the URL based on the root directory and the provided path
	if _, err := ls.resolvePath(path); err != nil {
		return "", err
	}
	fullPath := filepath.Join(ls.RootDirectory, path)
	// Assuming you are serving the files via HTTP
	// return fmt.Sprintf("http://yourdomain.com/%s", fullPath)

	return fullPath, nil
}

func (ls *LocalStorage) Open(path string) (*os.File, error) {
	fullPath, err := ls.resolvePath(path)
	if err != nil {
		return nil, err
	}
	return os.Open(fullPath)
}

func (ls *LocalStorage) Upload(file multipart.File, header *multipart.FileHeader, dir string) (*os.File, error) {
	if exists, _ := ls.Exists(dir); !exists {
		err := ls.CreateDirectory(dir)
		if err != nil {
			return nil, fmt.Errorf("could not create directory: %w", err)
		}
	}

	data, _ := io.ReadAll(file)
	err := ls.Write(path.Join(dir, header.Filename), data)
	if err != nil {
		return nil, fmt.Errorf("could not write file: %w", err)
	}

	if storedFile, err := ls.Open(path.Join(dir, header.Filename)); err != nil {
		return nil, fmt.Errorf("could not open file: %w", err)
	} else {
		defer func() {
			err := storedFile.Close()
			if err != nil {
				log.Println("Uploaded file could not be closed", "Error:", err)
			}
		}()
		return storedFile, nil
	}
}
