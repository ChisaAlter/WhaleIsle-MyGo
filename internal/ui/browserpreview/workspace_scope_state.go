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
// Empty Cwd deliberately means the stable shared/no-directory partition.
type WorkspaceOwner struct {
	Authority string
	Cwd       string
	Title     string
}

type browserWorkspace struct{ key, label string }

func makeBrowserWorkspace(owner WorkspaceOwner) (browserWorkspace, error) {
	if strings.TrimSpace(owner.Authority) == "" || len(owner.Authority) > 8192 || len(owner.Cwd) > 32768 || strings.ContainsRune(owner.Authority+owner.Cwd, 0) {
		return browserWorkspace{}, errors.New(browserChromeLabel("Browser 无法确定可信的工作区归属", "Browser cannot determine a trusted workspace owner"))
	}
	cwd := owner.Cwd
	if strings.TrimSpace(cwd) == "" {
		cwd = ""
	}
	kind, label := "shared", browserChromeLabel("共享浏览（无工作区）", "Shared browsing (no workspace)")
	if cwd != "" {
		kind = "workspace"
		label = strings.TrimSpace(owner.Title)
		if label == "" || label == owner.Cwd {
			label = cwd
		} else {
			label += " · " + cwd
		}
	}
	// JSON framing avoids delimiter collisions; only the digest reaches disk.
	encoded, _ := json.Marshal([]string{"browser-workspace-v1", owner.Authority, kind, cwd})
	sum := sha256.Sum256(encoded)
	return browserWorkspace{key: hex.EncodeToString(sum[:]), label: label}, nil
}

func (s browserWorkspace) needsNewTab(next browserWorkspace) bool {
	return s.key != "" && s.key != next.key
}
