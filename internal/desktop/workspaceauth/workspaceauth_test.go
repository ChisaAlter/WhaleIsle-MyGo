package workspaceauth

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestValidateAcceptsOrdinaryProject(t *testing.T) {
	root := t.TempDir()
	if err := Validate(root); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "src")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Validate(sub); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsEmptyMissingFile(t *testing.T) {
	if !errors.Is(Validate(""), ErrEmpty) {
		t.Fatal("empty")
	}
	if !errors.Is(Validate("   "), ErrEmpty) {
		t.Fatal("whitespace")
	}
	missing := filepath.Join(t.TempDir(), "nope")
	if err := Validate(missing); !errors.Is(err, ErrNotDirectory) {
		t.Fatalf("missing = %v", err)
	}
	f := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Validate(f); !errors.Is(err, ErrNotDirectory) {
		t.Fatalf("file = %v", err)
	}
}

func TestValidateRejectsFilesystemRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		vol := filepath.VolumeName(os.TempDir())
		if vol == "" {
			t.Skip("no volume")
		}
		root := vol + `\`
		if err := Validate(root); !errors.Is(err, ErrFilesystemRoot) {
			t.Fatalf("volume root = %v", err)
		}
	} else {
		if err := Validate("/"); !errors.Is(err, ErrFilesystemRoot) {
			t.Fatalf("/ = %v", err)
		}
	}
}

func TestValidateRejectsGitMetadataDirectory(t *testing.T) {
	root := t.TempDir()
	gitDir := filepath.Join(root, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Validate(gitDir); !errors.Is(err, ErrGitMetadata) {
		t.Fatalf(".git dir = %v", err)
	}
}

func TestIsHighRiskWorkspaceRoot(t *testing.T) {
	base := t.TempDir()
	anchorParent := filepath.Join(base, "users", "me")
	anchor := filepath.Join(anchorParent, "AppData")
	project := filepath.Join(anchorParent, "Documents", "proj")
	for _, dir := range []string{anchor, project} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	anchorReal, err := filepath.EvalSymlinks(anchor)
	if err != nil {
		t.Fatal(err)
	}
	anchorParentReal, err := filepath.EvalSymlinks(anchorParent)
	if err != nil {
		t.Fatal(err)
	}
	projectReal, err := filepath.EvalSymlinks(project)
	if err != nil {
		t.Fatal(err)
	}
	anchors := []string{anchorReal}
	if !isHighRiskWorkspaceRoot(anchorReal, anchors) {
		t.Fatal("anchor itself")
	}
	if !isHighRiskWorkspaceRoot(anchorParentReal, anchors) {
		t.Fatal("ancestor of anchor")
	}
	if isHighRiskWorkspaceRoot(projectReal, anchors) {
		t.Fatal("ordinary project under anchor parent")
	}
}

func TestValidateRejectsHighRiskWithInjectedAnchors(t *testing.T) {
	base := t.TempDir()
	anchor := filepath.Join(base, "AppData")
	project := filepath.Join(base, "Documents", "proj")
	for _, dir := range []string{anchor, project} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	anchorReal, err := filepath.EvalSymlinks(anchor)
	if err != nil {
		t.Fatal(err)
	}
	if err := validate(anchorReal, []string{anchorReal}); !errors.Is(err, ErrHighRisk) {
		t.Fatalf("anchor = %v", err)
	}
	if err := validate(project, []string{anchorReal}); err != nil {
		t.Fatalf("project = %v", err)
	}
}

func TestEscapesBaseAllowsDotDotNames(t *testing.T) {
	if escapesBase("..notes") {
		t.Fatal("..notes is a real name")
	}
	if !escapesBase("..") {
		t.Fatal(".. is traversal")
	}
	if !escapesBase(`..\outside`) && !escapesBase("../outside") {
		t.Fatal("../ traversal")
	}
}
