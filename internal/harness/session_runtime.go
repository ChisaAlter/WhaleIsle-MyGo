package harness

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"whaleisle/internal/startuptrace"
)

const sessionRuntimeManifestName = "host-session-runtime.json"
const sessionRuntimeHarnessCommit = "687f282e8625c6b8d11ff58211a90524be2e4317"
const sessionRuntimeRC4Commit = "504a01a9578abc2c6f1fb896b1dc322c9fe24f66"

type sessionRuntimeSafety struct {
	ID           string `json:"id"`
	BaseSHA256   string `json:"baseSHA256"`
	PatchSHA256  string `json:"patchSHA256"`
	OutputSHA256 string `json:"outputSHA256"`
}

var rc2SessionRuntimeSafety = sessionRuntimeSafety{
	ID:           "mygo-session-runtime-readiness-v1",
	BaseSHA256:   "b58fda1a53692c10f85f07ad1758cfdc5937062f721315e1b9d55427b34372d8",
	PatchSHA256:  "ed43fdb62635abf846038af45a1cb4a6d45f29147851ce8573c5dc8b41d08d5f",
	OutputSHA256: "8278107b486c56cd4519232f29e265f84f07943b51d469acd38cc0c6f694ec9b",
}

var rc4SessionRuntimeSafety = sessionRuntimeSafety{
	ID:           "mygo-session-runtime-readiness-v1",
	BaseSHA256:   "270c0b7a152a8f137ce255073c76cedce1aec68c2403a0033c17a57ee97ebd23",
	PatchSHA256:  "ed43fdb62635abf846038af45a1cb4a6d45f29147851ce8573c5dc8b41d08d5f",
	OutputSHA256: "00dd3a8aed5fe4f98f852d4d5a81d910cd1dbed01922cfa23a1004d7d364abb3",
}

func sessionRuntimeProfile(version string) (string, sessionRuntimeSafety, bool) {
	switch version {
	case "0.2.1-rc.2":
		return sessionRuntimeHarnessCommit, rc2SessionRuntimeSafety, true
	case "0.2.1-rc.4":
		return sessionRuntimeRC4Commit, rc4SessionRuntimeSafety, true
	default:
		return "", sessionRuntimeSafety{}, false
	}
}

var sessionRuntimeArtifacts = []string{
	"harness-upstream.json",
	"deepseek-harness/apps/cli/lib/bin.js",
	"deepseek-harness/packages/core/session/lib/index.js",
	"deepseek-harness/packages/session/session-persistence/lib/index.js",
	"deepseek-harness/packages/client/ui-visual-replies/lib/index.js",
}

// Validate the copied released runtime independently of the older source-build
// profiles. The visual-readiness boundary is still byte-pinned, not waived just
// because a newer Session reader is needed.
func validateStagedSessionRuntime(vendorDir string) error {
	return validateStagedSessionRuntimeWithReader(vendorDir, os.ReadFile)
}

// sessionRuntimeRegistryReader is scoped to one validation/Resolve call. Only
// the vocabulary read uses it; provenance, artifact hashes and Node identity
// retain their independent checks and existing digest policy.
type sessionRuntimeRegistryReader func(string) ([]byte, error)

func validateStagedSessionRuntimeWithReader(vendorDir string, read sessionRuntimeRegistryReader) error {
	endValidate := startuptrace.Begin("runtime.validate.session_runtime")
	defer endValidate()
	var manifest struct {
		SchemaVersion int    `json:"schemaVersion"`
		CLIVersion    string `json:"cliVersion"`
		HarnessPin    struct {
			Repo string `json:"repo"`
			Ref  string `json:"ref"`
			SHA  string `json:"sha"`
			Npm  string `json:"npm"`
		} `json:"harnessPin"`
		RequiredEventTypes []string             `json:"requiredEventTypes"`
		Artifacts          map[string]string    `json:"artifacts"`
		SafetyPatch        sessionRuntimeSafety `json:"safetyPatch"`
		Node               struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"node"`
	}
	data, err := os.ReadFile(filepath.Join(vendorDir, sessionRuntimeManifestName))
	if err != nil {
		return fmt.Errorf("compatible Host runtime provenance: %w", err)
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("compatible Host runtime provenance: %w", err)
	}
	commit, safety, supported := sessionRuntimeProfile(manifest.CLIVersion)
	if manifest.SchemaVersion != 1 || !supported ||
		manifest.HarnessPin.SHA != commit || manifest.HarnessPin.Ref != "dsh-v"+manifest.CLIVersion ||
		manifest.HarnessPin.Npm != manifest.CLIVersion || manifest.HarnessPin.Repo != "https://github.com/deepseek-ai/deepseek-harness.git" ||
		manifest.SafetyPatch != safety {
		return fmt.Errorf("compatible Host runtime has unsupported provenance or visual-readiness patch")
	}
	if manifest.Node.Path != "node/node.exe" {
		return fmt.Errorf("compatible Host runtime has no matching bundled Node")
	}
	if err := validateSessionRuntimeNode(vendorDir, filepath.Join(filepath.Dir(vendorDir), "node", "node.exe")); err != nil {
		return err
	}
	for _, required := range desktopSessionEventTypes {
		found := false
		for _, event := range manifest.RequiredEventTypes {
			found = found || event == required
		}
		if !found {
			return fmt.Errorf("compatible Host runtime did not verify required event %s", required)
		}
	}
	for _, artifact := range sessionRuntimeArtifacts {
		file := filepath.Join(vendorDir, filepath.FromSlash(artifact))
		if _, err := os.Stat(file); err != nil {
			return fmt.Errorf("compatible Host runtime artifact %s: %w", artifact, err)
		}
		digest, err := fileSHA256(file)
		if err != nil {
			return fmt.Errorf("compatible Host runtime artifact %s: %w", artifact, err)
		}
		if manifest.Artifacts[artifact] != digest {
			return fmt.Errorf("compatible Host runtime artifact changed: %s", artifact)
		}
		if artifact == sessionRuntimeArtifacts[4] && digest != safety.OutputSHA256 {
			return fmt.Errorf("compatible Host runtime visual-readiness gate is not the pinned build")
		}
	}
	return validateSessionRuntimeForLaunchWithReader(filepath.Join(vendorDir, "deepseek-harness"), read)
}

// Explicit overrides remain explicit, but cannot silently break the selected
// runtime's Node identity. A different location with identical bytes is valid.
func validateSessionRuntimeNode(vendorDir, node string) error {
	endHash := startuptrace.Begin("runtime.verify.node", node)
	defer endHash()
	data, err := os.ReadFile(filepath.Join(vendorDir, sessionRuntimeManifestName))
	if os.IsNotExist(err) {
		return nil // Older pinned source-build profiles own their existing checks.
	}
	if err != nil {
		return err
	}
	var manifest struct {
		Node struct {
			SHA256 string `json:"sha256"`
		} `json:"node"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return err
	}
	digest, err := fileSHA256(node)
	if err != nil {
		return fmt.Errorf("compatible Host runtime Node unavailable: %w", err)
	}
	if manifest.Node.SHA256 != digest {
		return fmt.Errorf("selected Node differs from the verified Host session runtime; use its matching bundled Node")
	}
	return nil
}

// fileDigest memoizes one content hash per (path, size, mtime) for this process.
// A single boot resolves the launcher up to five times and each resolution hashes
// the 93 MB bundled node.exe; re-reading it adds no evidence, and the key changes
// whenever the file does.
type fileDigest struct {
	size  int64
	mtime int64
	sum   string
}

var fileDigestCache sync.Map

func fileSHA256(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	key := filepath.Clean(path)
	stamp := fileDigest{size: info.Size(), mtime: info.ModTime().UnixNano()}
	if cached, ok := fileDigestCache.Load(key); ok {
		if prior, ok := cached.(fileDigest); ok && prior.size == stamp.size && prior.mtime == stamp.mtime {
			return prior.sum, nil
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	stamp.sum = hex.EncodeToString(sum[:])
	fileDigestCache.Store(key, stamp)
	return stamp.sum, nil
}

// The original desktop now writes these required v4 events. Format version 4
// alone does not establish reader compatibility: alpha.1 and rc.2 both use it.
var desktopSessionEventTypes = []string{
	"job/registered", "job/settled", "conversation/next-prompt-request",
	"team/control", "working-directory/change",
}

// Inspect the built registry used by persistence, never the TS source (which
// can be newer than a stale lib). The packaging tool also runs the real reader.
var builtSessionEventRegistry = regexp.MustCompile(`\bKNOWN_SESSION_EVENT_TYPES\s*=\s*new\s+Set\(\s*(\[[^\]]*\])\s*\)`)

// ValidateSessionRuntimeForLaunch refuses an old source launcher before it can
// attach to the shared desktop home. It does not inspect, migrate or rewrite
// sessions, and does not mark unknown events ignorable.
func ValidateSessionRuntimeForLaunch(harnessRoot string) error {
	return validateSessionRuntimeForLaunchWithReader(harnessRoot, os.ReadFile)
}

func validateSessionRuntimeForLaunchWithReader(harnessRoot string, read sessionRuntimeRegistryReader) error {
	file := filepath.Join(harnessRoot, "packages", "core", "session", "lib", "index.js")
	data, err := read(file)
	if err != nil {
		return fmt.Errorf("Host session reader unavailable at %s: %w; stage a complete compatible runtime with scripts/session-runtime.mjs", file, err)
	}
	match := builtSessionEventRegistry.FindSubmatch(data)
	var events []string
	if len(match) != 2 || json.Unmarshal(match[1], &events) != nil {
		return fmt.Errorf("cannot verify the built Host session event registry at %s; stage a complete compatible runtime with scripts/session-runtime.mjs", file)
	}
	known := make(map[string]bool, len(events))
	for _, event := range events {
		known[event] = true
	}
	for _, event := range desktopSessionEventTypes {
		if !known[event] {
			return fmt.Errorf("Host session reader is older than the shared desktop session schema (missing %s): %s; stage a complete supported runtime with scripts/session-runtime.mjs; session files were not changed", event, harnessRoot)
		}
	}
	return nil
}
