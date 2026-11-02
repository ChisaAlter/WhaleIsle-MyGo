package files

import (
	"os"
	"path/filepath"
	"testing"

	"whaleisle/internal/desktop/workspacefs"
)

// All paths are inert temporary fixtures; no executable is launched or opened
// by a Host. The forbidden fallback is represented by an instrumented callback.
func TestDeliveryPreviewNeverFallsBackToHost(t *testing.T) {
	root := t.TempDir()
	for name, body := range map[string]string{"program.exe": "inert test data", "readme.txt": "read only"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(root, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "secrets.txt"), []byte("inert fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	selected, preview, problem := "before.txt", "before", ""
	hostCalls := 0
	m := Model{WorkspaceRoot: root, SelectedFile: &selected, Preview: &preview, PreviewErr: &problem,
		OnOpenDoc:  func(string, workspacefs.DocKind, bool) { hostCalls++ },
		OnOpenWith: func(string, string) { hostCalls++ },
	}
	for _, path := range []string{"program.exe", "missing.txt", "readme.txt:12", "../outside.txt", ".git/secrets.txt", root} {
		if err := PreviewDeliveryPath(m, path); err == nil {
			t.Fatalf("unexpected preview success for %q", path)
		}
		if selected != "before.txt" || hostCalls != 0 {
			t.Fatalf("%q mutated another preview or invoked Host: selected=%q calls=%d", path, selected, hostCalls)
		}
	}
	if err := PreviewDeliveryPath(m, "readme.txt"); err != nil {
		t.Fatalf("valid text preview: %v", err)
	}
	if selected != "readme.txt" || preview != "read only" || hostCalls != 0 {
		t.Fatalf("wrong text preview selected=%q preview=%q Host=%d", selected, preview, hostCalls)
	}
}

func TestDeliveryPreviewKeepsDirtyTabsAndLeavesEditorReadOnly(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "readme.txt"), []byte("saved"), 0600); err != nil {
		t.Fatal(err)
	}
	selected, preview, draft, problem := "readme.txt", "saved", "unsaved", ""
	editing := true
	tabs := Tabs{}
	m := Model{BindingID: "local/session", WorkspaceRoot: root, Tabs: &tabs, SelectedFile: &selected, Preview: &preview, EditDraft: &draft, PreviewErr: &problem, Editing: &editing}
	tabs.Bind("other-host/session", root)
	if err := PreviewDeliveryPath(m, "readme.txt"); err == nil {
		t.Fatal("preview accepted a stale Files Host binding")
	}
	tabs.Bind(m.BindingID, root)
	prepareTabs(m)
	selected, preview, draft, editing = "readme.txt", "saved", "unsaved", true
	rememberTab(m, selected)
	if err := PreviewDeliveryPath(m, "readme.txt"); err == nil {
		t.Fatal("dirty tab was presented as a read-only saved preview")
	}
	if draft != "unsaved" || !editing {
		t.Fatal("preview refusal destroyed or altered an existing draft")
	}
	draft = preview
	capture(m)
	if err := PreviewDeliveryPath(m, "readme.txt"); err != nil {
		t.Fatal(err)
	}
	if editing || selectedTab(m).Editing {
		t.Fatal("delivery preview reopened an editable tab")
	}
}
