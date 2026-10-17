package workspaceauth

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"whaleisle/internal/paths"
)

var (
	// ErrEmpty means the path is missing or whitespace-only.
	ErrEmpty = errors.New("workspaceauth: workspace path is empty")
	// ErrNotDirectory means the path does not exist or is not a directory.
	ErrNotDirectory = errors.New("workspaceauth: workspace is not a directory")
	// ErrFilesystemRoot means the path is a volume or filesystem root.
	ErrFilesystemRoot = errors.New("workspaceauth: workspace cannot be a filesystem root")
	// ErrHighRisk means trusting the path would expose user home, app data, or dsh-home.
	ErrHighRisk = errors.New("workspaceauth: workspace path is not allowed")
	// ErrGitMetadata means the path is or traverses a .git metadata directory.
	ErrGitMetadata = errors.New("workspaceauth: workspace cannot be a .git directory")
)

// Validate checks that path is an acceptable boot workspace directory.
func Validate(path string) error {
	return validate(path, highRiskAnchorPaths())
}

func validate(raw string, anchors []string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ErrEmpty
	}
	resolved := filepath.Clean(trimmed)
	info, err := os.Stat(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrNotDirectory, resolved)
		}
		return fmt.Errorf("%w: %v", ErrNotDirectory, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%w: %s", ErrNotDirectory, resolved)
	}
	real, err := filepath.EvalSymlinks(resolved)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrNotDirectory, err)
	}
	real = filepath.Clean(real)
	if isFilesystemRoot(real) {
		return fmt.Errorf("%w: %s", ErrFilesystemRoot, real)
	}
	if hasGitDirSegment(real) {
		return fmt.Errorf("%w: %s", ErrGitMetadata, real)
	}
	if isHighRiskWorkspaceRoot(real, anchors) {
		return fmt.Errorf("%w: %s", ErrHighRisk, real)
	}
	return nil
}

func isFilesystemRoot(dir string) bool {
	dir = filepath.Clean(dir)
	if dir == "/" || dir == `\` {
		return true
	}
	vol := filepath.VolumeName(dir)
	if vol == "" {
		return false
	}
	root := vol + string(filepath.Separator)
	return dir == root || dir == vol+`/`
}

func highRiskAnchorPaths() []string {
	var candidates []string
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidates = append(candidates, home)
		if runtime.GOOS == "darwin" {
			candidates = append(candidates,
				filepath.Join(home, "Library"),
				filepath.Join(home, "Library", "Application Support"),
			)
		}
		candidates = append(candidates,
			filepath.Join(home, ".config"),
			filepath.Join(home, ".ssh"),
		)
	}
	for _, key := range []string{"APPDATA", "LOCALAPPDATA", "XDG_CONFIG_HOME", "XDG_DATA_HOME"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			candidates = append(candidates, value)
		}
	}
	dshHome := paths.DshHome()
	userData := paths.UserDataDir()
	candidates = append(candidates, dshHome, userData, filepath.Dir(dshHome))

	seen := make(map[string]struct{})
	var anchors []string
	for _, candidate := range candidates {
		resolved := filepath.Clean(candidate)
		real, err := filepath.EvalSymlinks(resolved)
		if err != nil {
			if _, statErr := os.Stat(resolved); statErr != nil {
				continue
			}
			real = resolved
		}
		key := identityKey(real)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		anchors = append(anchors, real)
	}
	return anchors
}

func isHighRiskWorkspaceRoot(real string, anchors []string) bool {
	for _, anchor := range anchors {
		if containedIn(real, anchor) {
			return true
		}
	}
	return false
}

func containedIn(root, candidate string) bool {
	fromRoot, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return !escapesBase(fromRoot)
}

func escapesBase(fromBase string) bool {
	if fromBase == "" {
		return false
	}
	if filepath.IsAbs(fromBase) {
		return true
	}
	return fromBase == ".." || strings.HasPrefix(fromBase, ".."+string(filepath.Separator))
}

func hasGitDirSegment(absPath string) bool {
	absPath = filepath.Clean(absPath)
	segments := strings.FieldsFunc(absPath, func(r rune) bool {
		return r == '/' || r == '\\'
	})
	for _, segment := range segments {
		if strings.EqualFold(segment, ".git") {
			return true
		}
	}
	return false
}

func identityKey(resolved string) string {
	if runtime.GOOS == "windows" {
		return strings.ToLower(resolved)
	}
	return resolved
}
