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

type highRiskAnchor struct {
	path           string
	rejectSubdirs  bool
	isLocalAppData bool
}

func resolveRealCandidate(candidate string) (string, bool) {
	resolved := filepath.Clean(candidate)
	real, err := filepath.EvalSymlinks(resolved)
	if err != nil {
		if _, statErr := os.Stat(resolved); statErr != nil {
			return "", false
		}
		real = resolved
	}
	return filepath.Clean(real), true
}

func resolveTempDirReal() string {
	tmp := strings.TrimSpace(os.TempDir())
	if tmp == "" {
		return ""
	}
	real, ok := resolveRealCandidate(tmp)
	if !ok {
		return ""
	}
	return real
}

func highRiskAnchorSpecs() []highRiskAnchor {
	var rawSpecs []highRiskAnchor
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		rawSpecs = append(rawSpecs, highRiskAnchor{path: home, rejectSubdirs: false})
		if runtime.GOOS == "darwin" {
			rawSpecs = append(rawSpecs,
				highRiskAnchor{path: filepath.Join(home, "Library"), rejectSubdirs: true},
				highRiskAnchor{path: filepath.Join(home, "Library", "Application Support"), rejectSubdirs: true},
			)
		}
		for _, sub := range []string{".config", ".ssh", ".gnupg", ".aws", ".kube"} {
			rawSpecs = append(rawSpecs, highRiskAnchor{path: filepath.Join(home, sub), rejectSubdirs: true})
		}
	}
	for _, key := range []string{"APPDATA", "LOCALAPPDATA", "XDG_CONFIG_HOME", "XDG_DATA_HOME"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			rawSpecs = append(rawSpecs, highRiskAnchor{
				path:           value,
				rejectSubdirs:  true,
				isLocalAppData: key == "LOCALAPPDATA",
			})
		}
	}
	dshHome := paths.DshHome()
	userData := paths.UserDataDir()
	rawSpecs = append(rawSpecs,
		highRiskAnchor{path: dshHome, rejectSubdirs: true},
		highRiskAnchor{path: userData, rejectSubdirs: true},
		highRiskAnchor{path: filepath.Dir(dshHome), rejectSubdirs: false},
	)

	seenIdx := make(map[string]int)
	var anchors []highRiskAnchor
	for _, candidate := range rawSpecs {
		real, ok := resolveRealCandidate(candidate.path)
		if !ok {
			continue
		}
		key := identityKey(real)
		if idx, exists := seenIdx[key]; exists {
			if candidate.rejectSubdirs {
				anchors[idx].rejectSubdirs = true
			}
			if !candidate.isLocalAppData {
				anchors[idx].isLocalAppData = false
			}
			continue
		}
		seenIdx[key] = len(anchors)
		anchors = append(anchors, highRiskAnchor{
			path:           real,
			rejectSubdirs:  candidate.rejectSubdirs,
			isLocalAppData: candidate.isLocalAppData,
		})
	}
	return anchors
}

func highRiskAnchorPaths() []string {
	specs := highRiskAnchorSpecs()
	anchors := make([]string, len(specs))
	for i, spec := range specs {
		anchors[i] = spec.path
	}
	return anchors
}

func isHighRiskWorkspaceRoot(real string, anchors []string) bool {
	if len(anchors) == 0 {
		return false
	}
	var homeKey, dshParentKey, localAppDataKey string
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if resolved, ok := resolveRealCandidate(home); ok {
			homeKey = identityKey(resolved)
		}
	}
	dshHome := paths.DshHome()
	userData := paths.UserDataDir()
	dshReal, dshOK := resolveRealCandidate(dshHome)
	userReal, userOK := resolveRealCandidate(userData)
	if parent := filepath.Dir(dshHome); parent != "" {
		if resolved, ok := resolveRealCandidate(parent); ok {
			key := identityKey(resolved)
			if (!dshOK || key != identityKey(dshReal)) && (!userOK || key != identityKey(userReal)) {
				dshParentKey = key
			}
		}
	}
	if localAppData := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); localAppData != "" {
		if resolved, ok := resolveRealCandidate(localAppData); ok {
			key := identityKey(resolved)
			if (!dshOK || key != identityKey(dshReal)) && (!userOK || key != identityKey(userReal)) {
				localAppDataKey = key
			}
		}
	}

	tempReal := resolveTempDirReal()
	realKey := identityKey(filepath.Clean(real))
	for _, anchor := range anchors {
		if containedIn(real, anchor) {
			return true
		}
		anchorKey := identityKey(filepath.Clean(anchor))
		if anchorKey == homeKey || anchorKey == dshParentKey {
			continue
		}
		if containedIn(anchor, real) {
			if anchorKey == localAppDataKey && tempReal != "" {
				tempKey := identityKey(tempReal)
				if tempKey != anchorKey && containedIn(anchor, tempReal) && realKey != tempKey && containedIn(tempReal, real) {
					continue
				}
			}
			return true
		}
	}
	return false
}

func containedIn(root, candidate string) bool {
	fromRoot, err := filepath.Rel(identityKey(filepath.Clean(root)), identityKey(filepath.Clean(candidate)))
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
