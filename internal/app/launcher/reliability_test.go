package launcherctl

import (
	"testing"

	"whaleisle/internal/config"
)

func TestControllerPanelStateDraftsAndWorkspaceTrim(t *testing.T) {
	deps := &stubSelectionDeps{}
	c := NewController(deps)

	cfg := &config.Config{
		Workspace:        "  /work/demo  ",
		QuitAfterStart:   true,
		CloseToTray:      true,
		AutoStartDesktop: true,
		AskOnUpdate:      false,
		DownloadRoute:    "cnb",
	}

	snap := c.PanelState(cfg, "")
	if snap.Tab != "home" || snap.Drafts.Workspace != "  /work/demo  " || !snap.Drafts.QuitAfterStart || !snap.Drafts.CloseToTray || !snap.Drafts.AutoStartDesktop || snap.Drafts.DownloadRoute != "cnb" {
		t.Fatalf("unexpected seeded PanelState: %+v", snap)
	}
	if seeded, val := c.CloseToTrayDraft(); !seeded || !val {
		t.Fatalf("CloseToTrayDraft = (%v, %v), want (true, true)", seeded, val)
	}
	if got := c.DownloadRouteOverride(); got != "cnb" {
		t.Fatalf("DownloadRouteOverride = %q, want cnb", got)
	}

	// CommitDrafts updates changed fields and advances WorkspaceGen.
	before := snap.Drafts
	next := before
	next.Workspace = "  /work/next  "
	next.QuitAfterStart = false
	next.CloseToTray = false
	next.AutoStartDesktop = false
	next.AskOnUpdate = true
	next.DownloadRoute = "github"
	next.RoutePopOpen = true
	next.ImportOverwrite = true
	next.ImportAttachments = true
	c.CommitDrafts(&before, next)

	got := c.PanelState(cfg, "")
	if got.Drafts.Workspace != "  /work/next  " || got.Drafts.QuitAfterStart || got.Drafts.CloseToTray || !got.Drafts.AskOnUpdate || got.Drafts.DownloadRoute != "github" || !got.Drafts.ImportOverwrite || !got.Drafts.ImportAttachments {
		t.Fatalf("CommitDrafts did not apply expected changes: %+v", got.Drafts)
	}

	// Stale WorkspaceGen commit must be ignored.
	staleBefore := snap.Drafts
	staleNext := staleBefore
	staleNext.Workspace = "/stale"
	c.CommitDrafts(&staleBefore, staleNext)
	if c.PanelState(cfg, "").Drafts.Workspace == "/stale" {
		t.Fatal("stale WorkspaceGen overwrote newer workspace draft")
	}

	// ResetWorkspaceDraft, SetStartError, and TryBeginStart mutate state cleanly.
	c.ResetWorkspaceDraft("  /reset/path \n")
	c.SetStartError(false, false, "")
	if ws, _, ok := c.TryBeginStart(); !ok || ws != "/reset/path" {
		t.Fatalf("TryBeginStart = (%q, %v), want trimmed /reset/path", ws, ok)
	}
	c.FinishStartPreparation(true, "boot failed", "", 0)
	after := c.PanelState(cfg, "")
	if after.StartBusy || !after.StartFailed || after.StartErr != "boot failed" {
		t.Fatalf("FinishStartPreparation not reflected: %+v", after)
	}
}
