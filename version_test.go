package uxid

import (
	"regexp"
	"testing"
)

// sabotage: set Version to "dev" in version.go; this test goes red.
func TestVersionIsSemver(t *testing.T) {
	semver := regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	if !semver.MatchString(Version) {
		t.Fatalf("Version = %q, want major.minor.patch", Version)
	}
}
