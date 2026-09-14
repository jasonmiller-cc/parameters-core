package version_test

import (
	"runtime"
	"strings"
	"testing"

	"github.com/jasonmiller-cc/parameters-core/pkg/version"
)

func TestGet(t *testing.T) {
	info := version.Get()

	if info.Version != version.Version {
		t.Errorf("Version = %q, want %q", info.Version, version.Version)
	}
	if info.OS != runtime.GOOS {
		t.Errorf("OS = %q, want %q", info.OS, runtime.GOOS)
	}
	if info.Arch != runtime.GOARCH {
		t.Errorf("Arch = %q, want %q", info.Arch, runtime.GOARCH)
	}
	if info.GoVersion == "" {
		t.Error("GoVersion should not be empty")
	}
}

func TestString(t *testing.T) {
	s := version.String()
	if !strings.Contains(s, version.Version) {
		t.Errorf("String() = %q, expected it to contain Version %q", s, version.Version)
	}
	if !strings.Contains(s, runtime.GOOS) || !strings.Contains(s, runtime.GOARCH) {
		t.Errorf("String() = %q, expected it to contain %s/%s", s, runtime.GOOS, runtime.GOARCH)
	}
}
