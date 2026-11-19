package directorypicker

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLocalListShowsOnlyDirectoriesInNameOrder(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"beta", "Alpha", ".hidden", "file.txt"} {
		path := filepath.Join(root, name)
		if strings.Contains(name, ".") && name != ".hidden" {
			if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	listing, err := LocalList(root)
	if err != nil {
		t.Fatalf("LocalList: %v", err)
	}
	var names []string
	for _, entry := range listing.Entries {
		names = append(names, entry.Name)
		if entry.Name == "file.txt" {
			t.Fatalf("a file is listed as a directory: %+v", entry)
		}
	}
	want := []string{".hidden", "Alpha", "beta"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("entries = %v, want %v", names, want)
	}
	if !listing.Entries[0].Hidden {
		t.Errorf("dot-prefixed directory is not flagged hidden")
	}
	if listing.Path != root {
		t.Errorf("path = %q, want %q", listing.Path, root)
	}
	last := listing.Crumbs[len(listing.Crumbs)-1]
	if last.Path != root {
		t.Errorf("last crumb = %+v, want the listed directory", last)
	}
	if runtime.GOOS == "windows" {
		if len(listing.Crumbs) < 2 || listing.Crumbs[0].Path != VolumeRoot {
			t.Errorf("crumbs do not start at the volume chooser: %+v", listing.Crumbs)
		}
	}
}

func TestLocalListRejectsAProcessRelativePath(t *testing.T) {
	_, err := LocalList("relative")
	if err == nil {
		t.Fatal("a process-relative path was listed")
	}
	var browse *Error
	if !errors.As(err, &browse) || browse.Kind != KindUnreadable {
		t.Fatalf("error = %v, want kind %q", err, KindUnreadable)
	}
}

func TestLocalListReportsAMissingDirectory(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent")
	_, err := LocalList(missing)
	var browse *Error
	if !errors.As(err, &browse) || browse.Kind != KindUnreadable || browse.Path != missing {
		t.Fatalf("error = %v, want unreadable at %q", err, missing)
	}
}

func TestLocalCreateDirectory(t *testing.T) {
	parent := t.TempDir()
	created, err := LocalCreateDirectory(parent, "workspace")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created != filepath.Join(parent, "workspace") {
		t.Fatalf("created = %q", created)
	}
	if st, err := os.Stat(created); err != nil || !st.IsDir() {
		t.Fatalf("created directory is absent: %v", err)
	}
	if _, err = LocalCreateDirectory(parent, "workspace"); err == nil {
		t.Fatal("creating an existing folder succeeded")
	} else {
		var browse *Error
		if !errors.As(err, &browse) || browse.Kind != KindExists {
			t.Fatalf("error = %v, want kind %q", err, KindExists)
		}
	}
	for _, name := range []string{"", "  ", ".", "..", `a\b`, "a/b"} {
		var browse *Error
		if _, err := LocalCreateDirectory(parent, name); !errors.As(err, &browse) || browse.Kind != KindCreateFailed {
			t.Errorf("segment %q was accepted: %v", name, err)
		}
	}
}

func TestUsableDirectory(t *testing.T) {
	dir := t.TempDir()
	got, err := UsableDirectory(dir)
	if err != nil {
		t.Fatalf("UsableDirectory: %v", err)
	}
	if got != dir {
		t.Fatalf("got %q, want %q", got, dir)
	}
	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"", "   ", VolumeRoot, file, filepath.Join(dir, "absent")} {
		var browse *Error
		if _, err := UsableDirectory(path); !errors.As(err, &browse) || browse.Kind != KindUnreadable {
			t.Errorf("path %q was accepted: %v", path, err)
		}
	}
}
