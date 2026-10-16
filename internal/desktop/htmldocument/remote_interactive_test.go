package htmldocument

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

type htmlHostFixture struct {
	files map[string]string
	paths []string
}

func (h *htmlHostFixture) Call(ctx context.Context, method string, args any, out any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	a := args.(map[string]any)
	if a["workspaceFileScopeId"] != "owner" {
		return fmt.Errorf("wrong source authority")
	}
	path := a["path"].(string)
	h.paths = append(h.paths, path)
	value, ok := h.files[path]
	if !ok {
		return fmt.Errorf("missing fixture %s", path)
	}
	result := map[string]any{"absolutePath": "/workspace/" + path, "version": "v1", "bytes": len(value)}
	if method == "workspaceFiles/readBytes" {
		result["data"] = []byte(value)
		result["offset"] = 0
		result["eof"] = true
	} else if method != "workspaceFiles/stat" {
		return fmt.Errorf("unexpected method %s", method)
	}
	raw, _ := json.Marshal(result)
	return json.Unmarshal(raw, out)
}
func TestHostHTMLInteractiveSnapshotSurvivesReadContext(t *testing.T) {
	host := &htmlHostFixture{files: map[string]string{"page.html": `<button onclick="this.textContent='changed'">Run</button><script src="page.js"></script>`, "page.js": "window.originalScript=true"}}
	live := true
	readCtx, cancel := context.WithCancel(context.Background())
	doc, err := OpenRemote(readCtx, host, "owner", "page.html", func() bool { return live })
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()
	cancel()
	if !doc.CanRunScripts() {
		t.Fatal("Host HTML did not expose interaction")
	}
	// Only the initial document is retained; the interaction gets a fresh scoped
	// resource read context and must never reread editor/file source changes.
	host.files["page.html"] = "changed saved file"
	prepared, _, err := doc.InteractiveSnapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prepared, "onclick") || !strings.Contains(prepared, base64.StdEncoding.EncodeToString([]byte(host.files["page.js"]))) || strings.Contains(prepared, "changed saved file") {
		t.Fatalf("lost source/scripts: %s", prepared)
	}
	live = false
	if _, _, err = doc.InteractiveSnapshot(context.Background()); err == nil {
		t.Fatal("revoked Host source still usable")
	}
}
