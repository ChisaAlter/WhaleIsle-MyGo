package chat

import (
	"os"
	"path/filepath"
	"testing"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/files"
)

func TestAssistantMarkdownImagesStayBetweenTheirAuthoredParagraphs(t *testing.T) {
	t.Setenv("WHALEISLE_USER_DATA", t.TempDir())
	SetAttachmentFetcher(nil)
	t.Cleanup(func() { SetAttachmentFetcher(nil) })
	msg := session.Message{ID: "ordered-markdown", Role: "assistant", Kind: session.MessageKindText, Body: "Before\n\n![one](attachment:first) ![two](attachment:second)\n\nMiddle\n[image] attachment:legacy\nAfter"}
	tt := messageTester(msg, 800, nil)
	last := float32(-1)
	for _, label := range []string{"Before", "[image] attachment:first", "[image] attachment:second", "Middle", "[image] attachment:legacy", "After"} {
		r, ok := tt.Find(label)
		if !ok || r.Y <= last {
			t.Fatalf("authored message order lost at %q: %v texts=%q", label, r, tt.Texts())
		}
		last = r.Y
	}
}

func TestAssistantMarkdownLocalLinkUsesExistingFilePreviewAndLineRequest(t *testing.T) {
	t.Setenv("WHALEISLE_USER_DATA", t.TempDir())
	root := t.TempDir()
	path := filepath.Join(root, "report.md")
	if err := os.WriteFile(path, []byte("first\nsecond\nthird\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var selected, preview, problem string
	tabs := files.Tabs{}
	model := files.Model{BindingID: "host/session", SessionID: "session", WorkspaceRoot: root, Tabs: &tabs, SelectedFile: &selected, Preview: &preview, PreviewErr: &problem}
	tabs.Bind(model.BindingID, root)
	var calls []string
	var openErr error
	msg := session.Message{ID: "link-message", Role: "assistant", Kind: session.MessageKindText, Body: "See [Report](report.md#L2-L3)."}
	tt := messageTester(msg, 800, func(m *Model) {
		m.Snapshot.SelectedID = "session"
		m.Snapshot.SelectedCwd = root
		m.OnOpenDeliverable = func(path string) { calls = append(calls, path); _, openErr = files.OpenPresentedPath(model, path) }
	})
	if selected != "" || len(calls) != 0 {
		t.Fatal("rendering opened the file")
	}
	if err := tt.Click("Report"); err != nil {
		t.Fatal(err)
	}
	if openErr != nil || selected != "report.md" || len(calls) != 1 || calls[0] != "report.md#L2" {
		t.Fatalf("file click=%q selected=%q err=%v", calls, selected, openErr)
	}
	workspace := tabs.Workspaces[filepath.Clean(root)]
	if workspace == nil || workspace.Active != "report.md" {
		t.Fatalf("preview tab was not selected: %+v", workspace)
	}
}
