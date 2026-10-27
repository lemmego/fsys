package fsys_test

import (
	"os/exec"
	"strings"
	"testing"
)

// TestTheRootIsTheStandardLibraryOnly pins the whole point of the driver
// subpackages.
//
// This package is imported by api/fs, api/providers/fs and media/provider,
// and through them by every application built on the framework. While the S3
// and GCS drivers lived here, all of them linked both cloud SDKs whether or
// not they had ever seen a bucket: measured on a trivial program, importing
// this package was the difference between 58 packages and 691, and between a
// 2.3MB binary and a 44MB one. In one real application it was half the
// binary, 263 of 961 packages, and several seconds of every rebuild.
//
// So a driver that needs a cloud SDK goes in its own package and registers
// itself from an init, the way database/sql drivers and cache/store/* do. An
// application links what it blank-imports and nothing else.
func TestTheRootIsTheStandardLibraryOnly(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatalf("listing dependencies: %v", err)
	}

	for _, dep := range strings.Fields(string(out)) {
		if dep == "github.com/lemmego/fsys" ||
			strings.HasPrefix(dep, "github.com/lemmego/fsys/") {
			continue
		}
		// A standard-library path has no dot in its first segment, which is
		// how the toolchain itself tells the two apart.
		first, _, _ := strings.Cut(dep, "/")
		if strings.Contains(first, ".") {
			t.Errorf("the root package depends on %s.\n"+
				"Only the standard library belongs here. A driver needing a "+
				"third-party SDK belongs in a subpackage that registers itself, "+
				"like fsys/s3 and fsys/gcs — otherwise every consumer of this "+
				"package links it.", dep)
		}
	}
}

// TestTheCloudDriversAreWhereTheirSDKsAreAllowed asserts the split is real
// rather than intended.
func TestTheCloudDriversAreWhereTheirSDKsAreAllowed(t *testing.T) {
	for _, tc := range []struct{ pkg, want string }{
		{"./s3", "github.com/aws/aws-sdk-go"},
		{"./gcs", "cloud.google.com/go/storage"},
	} {
		out, err := exec.Command("go", "list", "-deps", tc.pkg).Output()
		if err != nil {
			t.Fatalf("listing %s: %v", tc.pkg, err)
		}
		if !strings.Contains(string(out), tc.want) {
			t.Errorf("%s does not depend on %s, so either the driver is not "+
				"wired or this test is in the wrong place", tc.pkg, tc.want)
		}
	}
}
