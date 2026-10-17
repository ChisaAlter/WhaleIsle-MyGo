package phoneremote

import (
	"embed"
	"encoding/json"
	"io/fs"
	"strings"
)

//go:embed git-gate.cjs
var gitGateScript string

//go:embed mobile-draft-ui.js
var mobileDraftUI string

//go:embed mobile-draft-server.cjs
var mobileDraftServer string

//go:embed git-source
var originalGitSources embed.FS

func composedBridgeScript() string {
	sources := map[string]string{}
	_ = fs.WalkDir(originalGitSources, "git-source", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(name, ".js") {
			b, e := originalGitSources.ReadFile(name)
			if e != nil {
				return e
			}
			sources[strings.TrimPrefix(name, "git-source/")] = string(b)
		}
		return nil
	})
	encoded, _ := json.Marshal(sources)
	draftUI, _ := json.Marshal(mobileDraftUI)
	bridge := strings.ReplaceAll(bridgeScript, "__MYGO_GIT_SOURCES__", string(encoded))
	bridge = strings.ReplaceAll(bridge, "__MYGO_MOBILE_DRAFT_UI__", string(draftUI))
	return gitGateScript + "\n" + deviceActionsScript + "\n" + mobileDraftServer + "\n" + bridge
}

// SetGitAuthorizer checks the current desktop session for each authenticated
// original-daemon request. No root or permission is accepted from the phone.
func (r *Runtime) SetGitAuthorizer(authorize func(string, string) bool) {
	r.mu.Lock()
	r.authorizeGit = authorize
	r.mu.Unlock()
}
func (r *Runtime) RevokeGit() {
	r.mu.Lock()
	if r.reviewCancel != nil {
		r.reviewCancel()
	}
	child := r.child
	r.mu.Unlock()
	if child != nil {
		_ = r.sendControl(child, map[string]any{"action": "git-revoke"})
	}
}
func (r *Runtime) sendControl(child Child, value map[string]any) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	return writeControl(child, value)
}
