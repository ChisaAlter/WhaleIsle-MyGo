package paths

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"whaleisle/internal/config"
)

func problemOf(t *testing.T, err error) DataLocationProblem {
	t.Helper()
	var target *DataLocationError
	if !errors.As(err, &target) {
		t.Fatalf("error %v is not a DataLocationError", err)
	}
	return target.Problem
}

func TestValidateDataLocationChoiceRefusals(t *testing.T) {
	root := useTempConfigRoot(t)
	defaultDir := defaultUnder(root)
	current := filepath.Join(t.TempDir(), "current-data")
	if err := os.MkdirAll(current, 0o755); err != nil {
		t.Fatal(err)
	}
	install := filepath.Join(t.TempDir(), "Programs", "WhaleIsle-MyGo")
	if err := os.MkdirAll(install, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, config.AppDirName), 0o755); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name   string
		chosen string
		want   DataLocationProblem
	}{
		{"blank", "   ", DataLocationEmpty},
		{"relative", filepath.Join("some", "dir"), DataLocationRelative},
		{"existing file", file, DataLocationNotDirectory},
		{"equals current", current, DataLocationNested},
		{"inside current", filepath.Join(current, "child"), DataLocationNested},
		{"contains current", filepath.Dir(current), DataLocationNested},
		{"inside install", filepath.Join(install, "data"), DataLocationInsideInstall},
		{"contains install", filepath.Dir(install), DataLocationInsideInstall},
		{"contains record", ConfigRoot(), DataLocationContainsRecord},
		{"above record", filepath.Dir(ConfigRoot()), DataLocationContainsRecord},
	}
	for _, tc := range cases {
		err := ValidateDataLocationChoice(tc.chosen, current, install)
		if err == nil {
			t.Fatalf("%s: accepted %q", tc.name, tc.chosen)
		}
		if got := problemOf(t, err); got != tc.want {
			t.Fatalf("%s: problem = %s, want %s (%v)", tc.name, got, tc.want, err)
		}
	}

	accepted := []struct {
		name   string
		chosen string
	}{
		{"default", defaultDir},
		{"unrelated missing directory", filepath.Join(t.TempDir(), "new", "data")},
		{"beside current", filepath.Join(filepath.Dir(current), "other-data")},
	}
	for _, tc := range accepted {
		if err := ValidateDataLocationChoice(tc.chosen, current, install); err != nil {
			t.Fatalf("%s: refused %q: %v", tc.name, tc.chosen, err)
		}
	}
}

func TestValidateDataLocationChoiceIsCaseInsensitiveOnWindows(t *testing.T) {
	root := useTempConfigRoot(t)
	_ = root
	current := filepath.Join(t.TempDir(), "CurrentData")
	if err := os.MkdirAll(current, 0o755); err != nil {
		t.Fatal(err)
	}
	chosen := strings.ToUpper(current)
	err := ValidateDataLocationChoice(chosen, current, t.TempDir())
	if runtime.GOOS != "windows" {
		t.Skip("path comparison is case-sensitive here")
	}
	if err == nil {
		t.Fatalf("accepted the same directory spelled differently: %q", chosen)
	}
	if got := problemOf(t, err); got != DataLocationNested {
		t.Fatalf("problem = %s, want nested", got)
	}
}

func TestValidateDataLocationChoiceResolvesLinks(t *testing.T) {
	useTempConfigRoot(t)
	current := filepath.Join(t.TempDir(), "current-data")
	if err := os.MkdirAll(current, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "alias")
	if runtime.GOOS == "windows" {
		if err := os.Symlink(current, link); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
	} else if err := os.Symlink(current, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	err := ValidateDataLocationChoice(filepath.Join(link, "child"), current, t.TempDir())
	if err == nil {
		t.Fatal("accepted a path that resolves inside the current directory")
	}
	if got := problemOf(t, err); got != DataLocationNested {
		t.Fatalf("problem = %s, want nested", got)
	}
}

func TestValidateDataLocationChoiceRefusesUnwritableTarget(t *testing.T) {
	useTempConfigRoot(t)
	current := filepath.Join(t.TempDir(), "current-data")
	if err := os.MkdirAll(current, 0o755); err != nil {
		t.Fatal(err)
	}
	install := filepath.Join(t.TempDir(), "install")
	if err := os.MkdirAll(install, 0o755); err != nil {
		t.Fatal(err)
	}
	// A target under a regular file cannot be created, which is the portable
	// way to stand in for a read-only volume.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := ValidateDataLocationChoice(filepath.Join(blocker, "data"), current, install)
	if err == nil {
		t.Fatal("accepted a location that cannot be created")
	}
	if got := problemOf(t, err); got != DataLocationNotWritable {
		t.Fatalf("problem = %s, want not-writable", got)
	}
}

func TestValidateDataLocationChoiceRefusesWhenProbeFails(t *testing.T) {
	useTempConfigRoot(t)
	current := filepath.Join(t.TempDir(), "current-data")
	if err := os.MkdirAll(current, 0o755); err != nil {
		t.Fatal(err)
	}
	previous := probeDir
	probeDir = func(string) error { return errProbeRefused }
	t.Cleanup(func() { probeDir = previous })
	err := ValidateDataLocationChoice(filepath.Join(t.TempDir(), "data"), current, t.TempDir())
	if err == nil {
		t.Fatal("accepted a directory the write probe refused")
	}
	if got := problemOf(t, err); got != DataLocationNotWritable {
		t.Fatalf("problem = %s, want not-writable", got)
	}
}

// errProbeRefused stands in for an OS write failure in the probe seam.
var errProbeRefused = errors.New("probe refused")

func TestValidateDataLocationChoiceRefusesNetworkDrive(t *testing.T) {
	useTempConfigRoot(t)
	current := filepath.Join(t.TempDir(), "current-data")
	if err := os.MkdirAll(current, 0o755); err != nil {
		t.Fatal(err)
	}
	previous := remoteVolume
	remoteVolume = func(string) bool { return true }
	t.Cleanup(func() { remoteVolume = previous })

	err := ValidateDataLocationChoice(filepath.Join(t.TempDir(), "data"), current, t.TempDir())
	if err == nil {
		t.Fatal("accepted a network drive as the data location")
	}
	if got := problemOf(t, err); got != DataLocationRemoteVolume {
		t.Fatalf("problem = %s, want remote-volume", got)
	}

	// A local folder is still accepted while the check is on.
	remoteVolume = func(path string) bool { return path == "" }
	if err := ValidateDataLocationChoice(filepath.Join(t.TempDir(), "data2"), current, t.TempDir()); err != nil {
		t.Fatalf("refused a local folder: %v", err)
	}
}
