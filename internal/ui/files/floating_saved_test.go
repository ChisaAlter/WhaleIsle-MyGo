package files

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"whaleisle/internal/desktop/workspacefs"
)

func TestFloatingSavedReaderRefreshesWithoutDraft(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "note.md")
	if err := os.WriteFile(path, []byte("saved"), 0600); err != nil {
		t.Fatal(err)
	}
	draft := "unsaved draft"
	read := savedFloatingReader(Model{WorkspaceRoot: root, EditDraft: &draft}, "note.md")
	if got := read(context.Background()); got.err != nil || got.text != "saved" {
		t.Fatalf("%+v", got)
	}
	if err := os.WriteFile(path, []byte("saved again"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := read(context.Background()); got.err != nil || got.text != "saved again" || draft != "unsaved draft" {
		t.Fatalf("%+v draft=%q", got, draft)
	}
}
func TestFloatingSavedReaderHostAuthorityAndCancellation(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("local shadow"), 0600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	hostErr := errors.New("Host unavailable")
	read := savedFloatingReader(Model{WorkspaceRoot: root, ReadTextPageContext: func(ctx context.Context, path string, offset int64) (workspacefs.TextPage, error) {
		calls++
		return workspacefs.TextPage{}, hostErr
	}}, "note.md")
	if got := read(context.Background()); !errors.Is(got.err, hostErr) || got.text != "" {
		t.Fatalf("local fallback: %+v", got)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := read(ctx); !errors.Is(got.err, context.Canceled) || calls != 1 {
		t.Fatalf("canceled: %+v calls=%d", got, calls)
	}
}
