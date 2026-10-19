package browserpreview

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

// WorkspaceOwner comes from trusted desktop state, never from the guest URL.
// Authority must be a stable app/Host namespace, never a cookie, port or attempt
// ID. Cwd is opaque Host data: it is hashed verbatim; no desktop filesystem semantics are inferred.
// A ready authoritative Workspace membership provides Cwd; otherwise the
// selected Session remains isolated, including Sessions without a directory.
type WorkspaceOwner struct {
	Authority string
	Cwd       string
	Title     string
	SessionID string
	Ready     bool
	Err       error
}

type browserWorkspace struct{ key, label string }

func makeBrowserWorkspace(owner WorkspaceOwner) (browserWorkspace, error) {
	if owner.Err != nil {
		return browserWorkspace{}, owner.Err
	}
	if !owner.Ready {
		return browserWorkspace{}, errors.New(browserChromeLabel("工作区列表尚未就绪，请稍候重试", "The workspace list is not ready; try again shortly"))
	}
	if owner.SessionID == "" || len(owner.SessionID) > 8192 || strings.ContainsRune(owner.SessionID, 0) {
		return browserWorkspace{}, errors.New(browserChromeLabel("请选择 Browser 所属会话", "Select a session for this Browser"))
	}
	if strings.TrimSpace(owner.Authority) == "" || len(owner.Authority) > 8192 || len(owner.Cwd) > 32768 || strings.ContainsRune(owner.Authority+owner.Cwd, 0) {
		return browserWorkspace{}, errors.New(browserChromeLabel("Browser 无法确定可信的工作区归属", "Browser cannot determine a trusted workspace owner"))
	}
	cwd := owner.Cwd
	if strings.TrimSpace(cwd) == "" {
		cwd = ""
	}
	kind, identity := "session", owner.SessionID
	label := browserChromeLabel("会话浏览 · ", "Session browsing · ") + owner.SessionID
	if cwd != "" {
		kind, identity = "workspace", cwd
		label = strings.TrimSpace(owner.Title)
		if label == "" || label == owner.Cwd {
			label = cwd
		} else {
			label += " · " + cwd
		}
	}
	// JSON framing avoids delimiter collisions; only the digest reaches disk.
	encoded, _ := json.Marshal([]string{"browser-workspace-v1", owner.Authority, kind, identity})
	sum := sha256.Sum256(encoded)
	return browserWorkspace{key: hex.EncodeToString(sum[:]), label: label}, nil
}

func (s browserWorkspace) needsNewTab(next browserWorkspace) bool {
	return s.key != "" && s.key != next.key
}
