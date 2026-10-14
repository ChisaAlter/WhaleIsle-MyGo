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

const CurrentHostSourceCommit = "90465b67ec16765631d54c195af19d9ff3c3a770"

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
	SourceCommit      string `json:"sourceCommit,omitempty"`
	ArtifactsReady    bool   `json:"artifactsReady"`
	VisualReplyReader bool   `json:"visualReplyReader"`
	Project           bool   `json:"project"`
	Team              bool   `json:"team"`
	VisualReplies     bool   `json:"visualReplies"`
	Reason            string `json:"reason,omitempty"`
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
		SchemaVersion int               `json:"schemaVersion"`
		SourceCommit  string            `json:"sourceCommit"`
		Platform      string            `json:"platform"`
		Arch          string            `json:"arch"`
		BuildProfile  string            `json:"buildProfile"`
		Artifacts     map[string]string `json:"artifacts"`
	}
	if err = json.Unmarshal(raw, &m); err != nil || m.SchemaVersion != 1 || m.SourceCommit != CurrentHostSourceCommit || m.BuildProfile != "official" {
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
	c.VisualReplies = bridges.HTMLPreview && bridges.VisualReplyIsolation
	if !c.Project || !c.VisualReplies {
		c.Reason = "Matching Host artifacts are staged; desktop bridge, Project overlay, or HTML isolation is not ready"
	}
	return c
}
