package harness

import (
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

// ReadCurrentHostCapabilities reports only the pinned, successfully built
// assembly. It does not start Host or assume that a bridge is available.
// Call at startup/selection, not once per render: hashes intentionally fail
// closed when an installed artifact differs from the selected build.
func ReadCurrentHostCapabilities(vendorDir string, bridges CurrentHostBridges) CurrentHostCapabilities {
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
