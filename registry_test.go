package fsys_test

import (
	"strings"
	"testing"

	"github.com/lemmego/fsys"
)

// TestLocalAndMemoryAreAvailableWithNoImports, because they cost a consumer
// nothing and local is where every project starts.
func TestLocalAndMemoryAreAvailableWithNoImports(t *testing.T) {
	for _, driver := range []string{fsys.DRIVER_LOCAL, fsys.DRIVER_MEMORY} {
		disk, err := fsys.Open(driver, fsys.Settings{"path": t.TempDir()})
		if err != nil {
			t.Fatalf("opening %s: %v", driver, err)
		}
		if disk.Driver() != driver {
			t.Errorf("Driver() = %q, want %q", disk.Driver(), driver)
		}
	}
}

// TestAnUnregisteredDriverSaysWhatIsMissing.
//
// The usual cause is a missing blank import, not a typo — so an error saying
// only "unknown driver" sends somebody to their configuration, which is the
// one place that is right.
func TestAnUnregisteredDriverSaysWhatIsMissing(t *testing.T) {
	_, err := fsys.Open("s3", fsys.Settings{})
	if err == nil {
		t.Fatal("opening an unregistered driver succeeded")
	}
	for _, want := range []string{"blank import", "fsys/s3", "local"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not mention %q: %v", want, err)
		}
	}
}

// TestADuplicateRegistrationPanics, because the alternative is a disk opened
// silently by the wrong driver.
func TestADuplicateRegistrationPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("registering a driver twice did not panic")
		}
	}()
	fsys.Register(fsys.DRIVER_LOCAL, func(fsys.Settings) (fsys.FS, error) { return nil, nil })
}

// TestANilFactoryPanics for the same reason.
func TestANilFactoryPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("registering a nil factory did not panic")
		}
	}()
	fsys.Register("nilfactory", nil)
}

// TestLocalDefaultsToStorage, which is the path every scaffolded project uses.
func TestLocalDefaultsToStorage(t *testing.T) {
	disk, err := fsys.Open(fsys.DRIVER_LOCAL, nil)
	if err != nil {
		t.Fatalf("opening local with no settings: %v", err)
	}
	if disk == nil {
		t.Fatal("got a nil disk")
	}
}
