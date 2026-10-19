package files

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"whaleisle/internal/desktop/workspacefs"
)

// These are in-memory Host readers and temporary canaries, not a live Host or
// an account-backed end-to-end test. Navigation acceptance and preview failure
// are deliberately separate assertions.
func presentedHostFixture(t *testing.T, root string) (Model, <-chan struct{}) {
	t.Helper()
	var selected, preview, draft, problem string
	tabs := &Tabs{}
	done := make(chan struct{}, 8)
	m := Model{BindingID: "fixture-host/session-a", SessionID: "session-a", WorkspaceRoot: root,
		Tabs: tabs, SelectedFile: &selected, Preview: &preview, EditDraft: &draft, PreviewErr: &problem,
		notify: func() { done <- struct{}{} }}
	tabs.Bind(m.BindingID, root)
	t.Cleanup(tabs.Dispose)
	return m, done
}

func finishPresentedHostRead(t *testing.T, m Model, done <-chan struct{}) {
	t.Helper()
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	for tab := selectedTab(m); tab != nil && tab.readLoading; tab = selectedTab(m) {
		select {
		case <-done:
			syncTextRead(m)
		case <-deadline.C:
			t.Fatal("fixture read did not settle")
		}
	}
}

func TestPresentedHostTextIgnoresLocalNamesAndExistence(t *testing.T) {
	for _, tc := range []struct {
		name, local string
		missingRoot bool
		wholeText   bool
		failure     bool
	}{
		{name: "same-name-page", local: "LOCAL_CANARY"},
		{name: "missing-local-root", missingRoot: true},
		{name: "same-name-whole-text", local: "LOCAL_CANARY", wholeText: true},
		{name: "page-failure-keeps-local-hidden", local: "LOCAL_CANARY", failure: true},
		{name: "whole-text-failure-keeps-local-hidden", local: "LOCAL_CANARY", wholeText: true, failure: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if tc.missingRoot {
				root = filepath.Join(root, "host-only-workspace")
			}
			if tc.local != "" {
				if err := os.WriteFile(filepath.Join(root, "report.md"), []byte(tc.local), 0600); err != nil {
					t.Fatal(err)
				}
			}
			m, done := presentedHostFixture(t, root)
			calls := make(chan string, 2)
			read := func(ctx context.Context, path string) (string, error) {
				calls <- path
				if tc.failure {
					return "", errors.New("fixture Host read failed")
				}
				return "HOST_CONTENT", ctx.Err()
			}
			if tc.wholeText {
				m.ReadRemoteContext = read
			} else {
				m.ReadTextPageContext = func(ctx context.Context, path string, offset int64) (workspacefs.TextPage, error) {
					if offset != 0 {
						return workspacefs.TextPage{}, errors.New("unexpected fixture offset")
					}
					text, err := read(ctx, path)
					return workspacefs.TextPage{Text: text, EOF: true, Version: "host-v1"}, err
				}
			}
			handled, err := OpenPresentedPath(m, filepath.Join(root, "report.md"))
			if !handled || err != nil {
				t.Fatalf("Host navigation rejected by local existence: handled=%v err=%v", handled, err)
			}
			finishPresentedHostRead(t, m, done)
			select {
			case path := <-calls:
				if path != "report.md" {
					t.Fatalf("Host address changed: %q", path)
				}
			default:
				t.Fatal("same-name local file bypassed Host reader")
			}
			tab := selectedTab(m)
			if tab == nil || !tab.Remote || *m.Preview == "LOCAL_CANARY" {
				t.Fatal("Host tab lost its source authority")
			}
			if tc.failure {
				if *m.PreviewErr != "fixture Host read failed" || *m.Preview != "" {
					t.Fatalf("content failure fell back locally: %q / %q", *m.Preview, *m.PreviewErr)
				}
			} else if *m.Preview != "HOST_CONTENT" || *m.PreviewErr != "" {
				t.Fatalf("wrong preview source: %q / %q", *m.Preview, *m.PreviewErr)
			}
		})
	}
}

func TestPresentedHostTextContinuationKeepsHostCursor(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "report.md"), []byte("LOCAL_CANARY"), 0600); err != nil {
		t.Fatal(err)
	}
	m, done := presentedHostFixture(t, root)
	m.ReadTextPageContext = func(ctx context.Context, path string, offset int64) (workspacefs.TextPage, error) {
		if path != "report.md" {
			return workspacefs.TextPage{}, errors.New("wrong Host path")
		}
		switch offset {
		case 0:
			return workspacefs.TextPage{Text: "HOST_FIRST", Next: 7, Version: "host-v1"}, ctx.Err()
		case 7:
			return workspacefs.TextPage{Text: "HOST_SECOND", Separator: "\n", Next: 8, EOF: true, Version: "host-v1"}, ctx.Err()
		default:
			return workspacefs.TextPage{}, errors.New("local byte cursor leaked into Host paging")
		}
	}
	if handled, err := OpenPresentedPath(m, "report.md"); !handled || err != nil {
		t.Fatalf("open: %v %v", handled, err)
	}
	finishPresentedHostRead(t, m, done)
	beginNextTextPage(m)
	finishPresentedHostRead(t, m, done)
	if *m.Preview != "HOST_FIRST\nHOST_SECOND" || !selectedTab(m).Remote {
		t.Fatalf("continuation changed source: %q", *m.Preview)
	}
}
