// Package projectbridge adapts the pinned official Project desktop contract.
// It exposes no arbitrary command, executable, environment or file-read API.
package projectbridge

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"strings"
	"sync"
	"time"
	desktopgit "whaleisle/internal/desktop/git"
	"whaleisle/internal/directorypicker"
	"whaleisle/internal/processutil"
)

//go:embed runner.js source
var bundled embed.FS

const SourceCommit = "installed-0.3.13-20261102"

type Options struct {
	NodeBin, Home, Workspace string
	// OpenPath is a trusted native opener. The bridge resolves and authorizes first.
	OpenPath        func(string) error
	SelectDirectory func(context.Context, string) (string, error)
}
type Bridge struct {
	opts        Options
	mu          sync.Mutex
	workspaceMu sync.Mutex
	workspaces  map[string]*workspaceOperation
	selectionMu sync.Mutex
	readyMu     sync.Mutex
	readyNode   string
	readySize   int64
	readyMod    int64
	readyOK     bool
}

type workspaceOperation struct {
	done chan struct{}
}

var onReadyNodeProbe func()

func New(opts Options) *Bridge { return &Bridge{opts: opts} }

// Ready probes only module loading; it never starts Host or runs Git.
// A verified ready check for the exact Node binary (path, size, modtime) is
// memoized so repeated launcher/settings checks avoid redundant Node spawns.
func (b *Bridge) Ready(ctx context.Context) bool {
	if ctx != nil && ctx.Err() != nil {
		return false
	}
	if b.opts.NodeBin == "" || b.opts.Home == "" {
		return false
	}
	st, err := os.Stat(b.opts.NodeBin)
	if err != nil || !st.Mode().IsRegular() {
		return false
	}
	b.readyMu.Lock()
	cached := b.readyOK && b.readyNode == b.opts.NodeBin && b.readySize == st.Size() && b.readyMod == st.ModTime().UnixNano()
	b.readyMu.Unlock()
	if cached {
		return true
	}
	if onReadyNodeProbe != nil {
		onReadyNodeProbe()
	}
	result := b.call(ctx, map[string]any{"action": "ready"})
	ok := result["ok"] == true && result["available"] == true
	if ok {
		b.readyMu.Lock()
		b.readyNode, b.readySize, b.readyMod, b.readyOK = b.opts.NodeBin, st.Size(), st.ModTime().UnixNano(), true
		b.readyMu.Unlock()
	}
	return ok
}
func (b *Bridge) PrepareOverlay(ctx context.Context, sourceDir, profileDir, harnessRoot string) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	result := b.call(ctx, map[string]any{"action": "overlay", "sourceDir": sourceDir, "profileDir": profileDir, "harnessRoot": harnessRoot})
	if result["ok"] != true {
		return "", errors.New("Project overlay unavailable")
	}
	overlay, _ := result["overlayFile"].(string)
	if overlay == "" {
		return "", errors.New("Project overlay unavailable")
	}
	return overlay, nil
}

// Dispatch authorizes write targets before acquiring the shared repository owner.
// Reads and unrelated repositories remain independent. Durable official receipts
// make repeated prepare-workspace calls idempotent across restarts.
func (b *Bridge) Dispatch(ctx context.Context, payload map[string]any) map[string]any {
	action, _ := payload["action"].(string)
	switch action {
	case "select-directory", "canonicalize", "repository", "prepare-workspace", "check-workspace", "snapshot", "cleanup-workspace", "check-directory", "baseline", "changes", "revert", "resolve-path", "pull-request", "open-path":
	default:
		return failure("unsupported Project operation")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if action == "select-directory" {
		if !b.selectionMu.TryLock() {
			return failure("Project directory selection already open")
		}
		defer b.selectionMu.Unlock()
		if b.opts.SelectDirectory == nil {
			return failure("unsupported: native Project directory picker unavailable")
		}
	}
	request := make(map[string]any, len(payload))
	for k, v := range payload {
		request[k] = v
	}
	if action == "prepare-workspace" || action == "cleanup-workspace" {
		release, err := b.acquireWorkspaceOperation(ctx, request)
		if err != nil {
			return failure("Project workspace operation was cancelled")
		}
		defer release()
	}
	if action == "select-directory" {
		suggested, _ := request["workingDirectory"].(string)
		selected, err := b.opts.SelectDirectory(ctx, suggested)
		if errors.Is(err, directorypicker.ErrCancelled) || (err == nil && selected == "") {
			return map[string]any{"ok": false, "cancelled": true, "error": "Project directory selection cancelled"}
		}
		if err != nil {
			return failure("Project directory selection or workspace registration failed")
		}
		if ctx.Err() != nil {
			return failure("Project directory selection interrupted")
		}
		request["workingDirectory"] = selected
	}
	if action == "open-path" {
		if b.opts.OpenPath == nil {
			return failure("unsupported: native Project opener unavailable")
		}
		request["action"] = "resolve-path"
	}
	if projectWriteAction(request) {
		preflight := make(map[string]any, len(request)+1)
		for k, v := range request {
			preflight[k] = v
		}
		preflight["action"], preflight["operation"] = "write-target", action
		target := b.call(ctx, map[string]any{"action": "dispatch", "payload": preflight})
		if target["ok"] != true {
			return target
		}
		root, _ := target["root"].(string)
		if root == "" {
			return b.call(ctx, map[string]any{"action": "dispatch", "payload": request})
		}
		scope := desktopgit.WorktreeWrite
		if target["scope"] == "references" {
			scope = desktopgit.ReferenceWrite
		}
		owned, release, err := desktopgit.AcquireWrite(ctx, root, scope)
		if err != nil {
			return failure("Project Git write admission failed or was cancelled")
		}
		defer release()
		ctx = owned
	}
	result := b.call(ctx, map[string]any{"action": "dispatch", "payload": request})
	if action == "open-path" && result["ok"] == true {
		target, _ := result["canonicalPath"].(string)
		if target == "" {
			return failure("Project path unavailable")
		}
		if err := b.opts.OpenPath(target); err != nil {
			return failure("Project file could not be opened")
		}
	}
	return result
}

// Receipts also exist for non-Git directories. Serialize only this binding;
// different Project workstreams and read-only calls do not share the gate.
func (b *Bridge) acquireWorkspaceOperation(ctx context.Context, request map[string]any) (func(), error) {
	projectID, _ := request["projectId"].(string)
	workstreamID, _ := request["workstreamId"].(string)
	key := projectID + "\x00" + workstreamID
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		b.workspaceMu.Lock()
		if b.workspaces == nil {
			b.workspaces = make(map[string]*workspaceOperation)
		}
		prior := b.workspaces[key]
		if prior == nil {
			operation := &workspaceOperation{done: make(chan struct{})}
			b.workspaces[key] = operation
			b.workspaceMu.Unlock()
			var once sync.Once
			return func() {
				once.Do(func() {
					b.workspaceMu.Lock()
					delete(b.workspaces, key)
					close(operation.done)
					b.workspaceMu.Unlock()
				})
			}, nil
		}
		b.workspaceMu.Unlock()
		select {
		case <-prior.done:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func projectWriteAction(request map[string]any) bool {
	switch request["action"] {
	case "prepare-workspace":
		return request["mode"] == "worktree"
	case "cleanup-workspace", "baseline":
		return true
	case "changes":
		end, _ := request["end"].(string)
		return end == ""
	case "revert":
		checkOnly, _ := request["checkOnly"].(bool)
		return !checkOnly
	default:
		return false
	}
}
func failure(message string) map[string]any {
	return map[string]any{"ok": false, "available": false, "error": message}
}

type limitedBuffer struct {
	bytes.Buffer
	limit int
}

func (w *limitedBuffer) Write(p []byte) (int, error) {
	if len(p) > w.limit-w.Len() {
		return 0, errors.New("Project response too large")
	}
	return w.Buffer.Write(p)
}
func (b *Bridge) call(parent context.Context, envelope map[string]any) map[string]any {
	if parent == nil {
		parent = context.Background()
	}
	if b.opts.NodeBin == "" || b.opts.Home == "" {
		return failure("Project desktop bridge unavailable")
	}
	sources := map[string]string{}
	err := fs.WalkDir(bundled, "source", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(name, ".js") {
			raw, e := bundled.ReadFile(name)
			if e != nil {
				return e
			}
			sources[strings.TrimPrefix(name, "source/")] = string(raw)
		}
		return nil
	})
	if err != nil {
		return failure("Project desktop sources unavailable")
	}
	envelope["sources"], envelope["home"], envelope["workspace"] = sources, b.opts.Home, b.opts.Workspace
	input, err := json.Marshal(envelope)
	if err != nil {
		return failure("invalid Project request")
	}
	script, _ := bundled.ReadFile("runner.js")
	ctx, cancel := context.WithTimeout(parent, 3*time.Minute)
	defer cancel()
	cmd := processutil.CommandContext(ctx, b.opts.NodeBin, "-e", string(script))
	processutil.BackgroundTree(cmd)
	cmd.Stdin = bytes.NewReader(input)
	// Prevent ambient Node injection and inherited control-plane credentials.
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		upper := strings.ToUpper(key)
		if upper == "NODE_OPTIONS" || upper == "NODE_PATH" || upper == "ELECTRON_RUN_AS_NODE" ||
			upper == "DSHD_HOME" || upper == "DSH_HOME" ||
			upper == "GIT_DIR" || upper == "GIT_WORK_TREE" || upper == "GIT_INDEX_FILE" ||
			upper == "GIT_COMMON_DIR" || upper == "GIT_OBJECT_DIRECTORY" || upper == "GIT_ALTERNATE_OBJECT_DIRECTORIES" ||
			strings.HasPrefix(upper, "MAGPIE_") || strings.HasPrefix(upper, "DSH_DESKTOP_INSTALL_") {
			continue
		}
		cmd.Env = append(cmd.Env, entry)
	}
	cmd.Env = append(cmd.Env, "DSHD_HOME="+b.opts.Home, "DSH_HOME="+b.opts.Home)
	output := &limitedBuffer{limit: 4 * 1024 * 1024}
	cmd.Stdout = output
	if err = cmd.Run(); err != nil {
		return failure("Project desktop operation failed or timed out")
	}
	var result map[string]any
	if json.Unmarshal(output.Bytes(), &result) != nil || result == nil {
		return failure("invalid Project desktop response")
	}
	return result
}
