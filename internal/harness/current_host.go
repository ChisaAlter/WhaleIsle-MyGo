package harness

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const CurrentHostReadinessGuardID = "mygo-visual-readiness-v1"
const CurrentHostReadinessPatchSHA256 = "87b97484bbed6adc2d727e49922766af96579a6b4337a44da799cc993b13d3b3"

// Retain the original default; selecting e9 artifacts is explicit and never upgrades an installed runtime.
const CurrentHostSourceCommit = "90465b67ec16765631d54c195af19d9ff3c3a770"
const CurrentHostE9SourceCommit = "e9f584be6d3b491ce777bb6823f30bc2bbc6f195"

func supportedCurrentHostSource(commit string) bool {
	return commit == CurrentHostSourceCommit || commit == CurrentHostE9SourceCommit
}

// Bridge readiness must come from the running desktop implementation, not a
// server setting or a package's shared version number. Overlay readiness means
// the original Project adapter successfully resolved peers in this profile.
type CurrentHostBridges struct {
	Project              bool
	ProjectOverlay       bool
	HTMLPreview          bool
	VisualReplyIsolation bool
}

type CurrentHostCapabilities struct {
	SourceCommit        string `json:"sourceCommit,omitempty"`
	ArtifactsReady      bool   `json:"artifactsReady"`
	ReadinessGuardReady bool   `json:"readinessGuardReady"`
	VisualReplyReader   bool   `json:"visualReplyReader"`
	Project             bool   `json:"project"`
	Team                bool   `json:"team"`
	VisualReplies       bool   `json:"visualReplies"`
	Reason              string `json:"reason,omitempty"`
}

var currentHostArtifacts = []string{
	"deepseek-harness/apps/cli/lib/bin.js", "deepseek-harness/apps/web/dist/index.html",
	"deepseek-harness/packages/experimental/agent-team/lib/index.js",
	"deepseek-harness/packages/experimental/agent-team-profile/lib/index.js",
	"deepseek-harness/packages/experimental/tool-agent-team/lib/index.js",
	"deepseek-harness/packages/experimental/client-ui-agent-team/lib/index.js",
	"deepseek-harness/packages/client/ui-visual-replies/lib/index.js",
	"deepseek-harness/packages/api/session-controller/lib/index.js",
	"deepseek-harness/packages/client/ui-visual-replies/lib/client.js",
	"dsh-project/lib/index.js", "dsh-project/lib/transport.js", "dsh-project/client/client.js",
}

// The installed rc.2 runtime ships no source-build manifest. Its own staged-
// session proof is at least as strict: the rc.2 pins and artifact hashes in
// host-session-runtime.json (validated by validateStagedSessionRuntime, which
// already requires the visual-readiness output hash), the readiness record that
// names the same pinned triple, the gate marker inside that bundle, and the
// Project package the desktop bridge stages into the profile.
const installedReadinessRecordName = "session-runtime-readiness.json"
const installedVisualRepliesBundle = "deepseek-harness/packages/client/ui-visual-replies/lib/index.js"

// installedVisualReplyGateMarkers are the strings the rc.2 readiness stage adds
// to the released bundle (round146 §10.2). A hash alone proves the bytes are the
// pinned build; the markers prove we know which build that is.
var installedVisualReplyGateMarkers = []string{
	"MyGo visual reply readiness gate v1",
	"/desktop/html-preview/status",
}

// installedProjectArtifacts are the files ProjectBridge.PrepareOverlay needs. No
// manifest covers dsh-project, so they are checked for presence here; the bridge
// fails closed on its own when the staged copy is incomplete.
var installedProjectArtifacts = []string{
	"dsh-project/lib/index.js",
	"dsh-project/lib/transport.js",
	"dsh-project/client/client.js",
	"dsh-project/package.json",
}

// readInstalledRuntimeCapabilities proves an installed rc.2 runtime. Every check
// fails closed with a Reason; nothing here trusts a self-declared file alone.
func readInstalledRuntimeCapabilities(vendorDir string, bridges CurrentHostBridges) CurrentHostCapabilities {
	c := CurrentHostCapabilities{}
	fail := func(reason string) CurrentHostCapabilities { c.Reason = reason; return c }
	// 1. The staged session runtime: rc.2 pins, bundled Node, its five hashed
	// artifacts (ui-visual-replies must be the pinned readiness output) and the
	// desktop session event vocabulary. Reused, never duplicated.
	if err := validateStagedSessionRuntime(vendorDir); err != nil {
		return fail("Installed Host runtime does not prove the pinned rc.2 assembly: " + err.Error())
	}
	// 2. The readiness record must name the same pinned triple.
	raw, err := os.ReadFile(filepath.Join(vendorDir, installedReadinessRecordName))
	if err != nil {
		return fail("Installed Host readiness record is missing: " + installedReadinessRecordName)
	}
	var record sessionRuntimeSafety
	if err := json.Unmarshal(raw, &record); err != nil || record != rc2SessionRuntimeSafety {
		return fail("Installed Host readiness record does not pin the visual-readiness gate")
	}
	// 3. The hashed bundle must still carry the gate itself.
	bundle, err := os.ReadFile(filepath.Join(vendorDir, filepath.FromSlash(installedVisualRepliesBundle)))
	if err != nil {
		return fail("Installed Host visual-replies bundle is missing")
	}
	for _, marker := range installedVisualReplyGateMarkers {
		if !bytes.Contains(bundle, []byte(marker)) {
			return fail("Installed Host visual-replies bundle lacks the readiness gate: " + marker)
		}
	}
	// 4. The Project package the desktop bridge stages into the profile.
	for _, name := range installedProjectArtifacts {
		if st, err := os.Stat(filepath.Join(vendorDir, filepath.FromSlash(name))); err != nil || st.IsDir() {
			return fail("Installed Host Project package is incomplete: " + name)
		}
	}
	// Provenance is the rc.2 release commit the session runtime pins; the source
	// build's sourceCommit is not claimed here.
	c.SourceCommit = sessionRuntimeHarnessCommit
	c.ArtifactsReady = true
	c.VisualReplyReader = true
	c.ReadinessGuardReady = true
	c.Project = bridges.Project && bridges.ProjectOverlay
	c.Team = c.Project
	c.VisualReplies = c.ReadinessGuardReady && bridges.HTMLPreview && bridges.VisualReplyIsolation
	if !c.Project || !c.VisualReplies {
		c.Reason = "Installed Host artifacts are staged; desktop bridge, Project overlay, or HTML isolation is not ready"
	}
	return c
}

// ReadCurrentHostCapabilities reports only the pinned, successfully built
// assembly. It does not start Host or assume that a bridge is available.
// Call at startup/selection, not once per render: hashes intentionally fail
// closed when an installed artifact differs from the selected build.
//
// A source build is proven by current-host-build.json. Without it, an installed
// rc.2 runtime is proven by its own staged-session record (round146 §10.5), so
// Project mounts on the installed runtime too.
func ReadCurrentHostCapabilities(vendorDir string, bridges CurrentHostBridges) CurrentHostCapabilities {
	if _, err := os.Stat(filepath.Join(vendorDir, "current-host-build.json")); os.IsNotExist(err) {
		return readInstalledRuntimeCapabilities(vendorDir, bridges)
	}
	c := CurrentHostCapabilities{}
	fail := func(reason string) CurrentHostCapabilities { c.Reason = reason; return c }
	raw, err := os.ReadFile(filepath.Join(vendorDir, "current-host-build.json"))
	if err != nil {
		return fail("Matching current Host build is not staged")
	}
	var m struct {
		SchemaVersion int    `json:"schemaVersion"`
		SourceCommit  string `json:"sourceCommit"`
		Platform      string `json:"platform"`
		Arch          string `json:"arch"`
		SafetyPatch   struct {
			ID           string `json:"id"`
			PatchSHA256  string `json:"patchSHA256"`
			SourceSHA256 string `json:"sourceSHA256"`
		} `json:"safetyPatch"`
		BuildProfile string            `json:"buildProfile"`
		Artifacts    map[string]string `json:"artifacts"`
	}
	if err = json.Unmarshal(raw, &m); err != nil || m.SchemaVersion != 1 || !supportedCurrentHostSource(m.SourceCommit) || m.BuildProfile != "official" {
		return fail("Current Host build provenance is unsupported")
	}
	platform := runtime.GOOS
	if platform == "windows" {
		platform = "win32"
	}
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}
	if m.Platform != platform || m.Arch != arch {
		return fail("Current Host artifacts were built for a different platform or architecture")
	}
	for _, name := range currentHostArtifacts {
		data, err := os.ReadFile(filepath.Join(vendorDir, filepath.FromSlash(name)))
		if err != nil {
			return fail(fmt.Sprintf("Current Host artifact missing: %s", name))
		}
		sum := sha256.Sum256(data)
		if m.Artifacts[name] != hex.EncodeToString(sum[:]) {
			return fail(fmt.Sprintf("Current Host artifact changed: %s", name))
		}
	}
	c.SourceCommit = m.SourceCommit
	c.ArtifactsReady = true
	c.VisualReplyReader = true
	c.Project = bridges.Project && bridges.ProjectOverlay
	c.Team = c.Project
	c.ReadinessGuardReady = m.SafetyPatch.ID == CurrentHostReadinessGuardID && m.SafetyPatch.PatchSHA256 == CurrentHostReadinessPatchSHA256 && m.SafetyPatch.SourceSHA256 == "8c73f32eb14e9ba46d3712bcbc569b2effe74c28bc60fbfa299e08c920bf2812"
	c.VisualReplies = c.ReadinessGuardReady && bridges.HTMLPreview && bridges.VisualReplyIsolation
	if !c.Project || !c.VisualReplies {
		c.Reason = "Matching Host artifacts are staged; desktop bridge, Project overlay, or HTML isolation is not ready"
	}
	return c
}

// ValidateCurrentHostForLaunch prevents a saved enabled=true from executing in
// an old current-source runtime that predates the per-operation readiness gate.
// Legacy customized runtimes without visual-replies are left unchanged.
func ValidateCurrentHostForLaunch(harnessRoot string) error {
	vendorDir := filepath.Dir(harnessRoot)
	_, manifestErr := os.Stat(filepath.Join(vendorDir, "current-host-build.json"))
	_, visualErr := os.Stat(filepath.Join(harnessRoot, "packages", "client", "ui-visual-replies", "package.json"))
	if os.IsNotExist(manifestErr) {
		if _, err := os.Stat(filepath.Join(vendorDir, sessionRuntimeManifestName)); err == nil {
			return validateStagedSessionRuntime(vendorDir)
		}
	}
	if os.IsNotExist(manifestErr) && os.IsNotExist(visualErr) {
		return nil
	}
	c := ReadCurrentHostCapabilities(vendorDir, CurrentHostBridges{})
	if !c.ArtifactsReady || !c.ReadinessGuardReady {
		return fmt.Errorf("current Host cannot start: rebuild with the MyGo visual-readiness safety patch; %s", c.Reason)
	}
	return nil
}
