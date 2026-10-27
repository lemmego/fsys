package fsys

import (
	"fmt"
	"sort"
	"sync"
)

// Settings are a disk's configuration, as the application wrote it down.
//
// A map of strings rather than a struct per driver, because the values come
// from configuration and arrive as strings anyway, and because a driver
// written outside this module has to be able to read keys this one has never
// heard of.
type Settings map[string]string

// Get returns a setting, or "".
func (s Settings) Get(key string) string { return s[key] }

// Factory opens a disk from its settings.
type Factory func(Settings) (FS, error)

var (
	mu       sync.RWMutex
	builders = map[string]Factory{}
)

// Register makes a driver available to [Open], under a name a disk's
// configuration can select.
//
// Drivers register themselves from an init in their own package, the way
// database/sql drivers do, so that a deployment links only the storage it
// actually uses. That is the whole point of the arrangement: this root
// package is the standard library only, and serving S3 and GCS from it meant
// every consumer — every application, and api/fs along with it — linked both
// cloud SDKs whether or not it had ever seen a bucket. Measured on a trivial
// program, that was the difference between 58 packages and 691.
//
// It panics on a duplicate or a nil factory, which is what database/sql does
// and for the same reason: both are programming errors visible at startup,
// and the alternative is a disk silently opened by the wrong driver.
func Register(driver string, factory Factory) {
	mu.Lock()
	defer mu.Unlock()

	if factory == nil {
		panic("fsys: Register called with a nil factory for driver " + driver)
	}
	if _, taken := builders[driver]; taken {
		panic("fsys: driver " + driver + " is already registered")
	}
	builders[driver] = factory
}

// Open builds a disk from its driver name and settings.
//
// An unknown driver names the ones that are registered, because the usual
// cause is a missing blank import rather than a typo — and an error that says
// only "unknown driver s3" sends somebody looking in their configuration,
// which is the one place that is right.
func Open(driver string, settings Settings) (FS, error) {
	mu.RLock()
	factory, known := builders[driver]
	mu.RUnlock()

	if !known {
		return nil, fmt.Errorf(
			"fsys: no driver %q is registered (have: %v). A driver registers "+
				"itself when its package is imported, so this usually means a "+
				"missing blank import such as _ \"github.com/lemmego/fsys/s3\"",
			driver, Drivers())
	}
	return factory(settings)
}

// Drivers names the registered drivers, sorted.
func Drivers() []string {
	mu.RLock()
	defer mu.RUnlock()

	names := make([]string, 0, len(builders))
	for name := range builders {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// local and memory register here rather than in subpackages of their own,
// because neither costs a consumer anything: both are the standard library,
// and local is the default every project starts on.
func init() {
	Register(DRIVER_LOCAL, func(s Settings) (FS, error) {
		path := s.Get("path")
		if path == "" {
			path = "./storage"
		}
		return NewLocalStorage(path), nil
	})

	Register(DRIVER_MEMORY, func(Settings) (FS, error) {
		return NewMemoryStorage(), nil
	})
}
