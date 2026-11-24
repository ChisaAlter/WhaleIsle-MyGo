package sidebar

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

func TestCossSidebarModalDialogsCursorAudit(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })

	draft := "Session Title"
	noop := func() {}
	cases := []struct {
		name  string
		paint func(*ui.Context)
	}{
		{
			name: "rename",
			paint: func(c *ui.Context) {
				sessionRenameDialog(c, Model{
					ThemeID:         "deepseek.light",
					RenameSessionID: "s1",
					RenameDraft:     &draft,
					OnCommitRename:  func(string, string) {},
					OnCancelRename:  noop,
				})
			},
		},
		{
			name: "archive-confirm",
			paint: func(c *ui.Context) {
				sessionArchiveConfirmDialog(c, Model{
					ThemeID: "deepseek.light",
					ArchiveConfirm: &ArchiveConfirmState{
						SessionID:   "s1",
						Title:       "Active Session",
						TurnRunning: true,
						OnConfirm:   func(string) {},
						OnCancel:    noop,
					},
				})
			},
		},
		{
			name: "delete-confirm",
			paint: func(c *ui.Context) {
				sessionDeleteConfirmDialog(c, Model{
					ThemeID: "deepseek.light",
					DeleteConfirm: &DeleteConfirmState{
						SessionID: "s1",
						Title:     "Archived Session",
						OnConfirm: func(string) {},
						OnCancel:  noop,
					},
				})
			},
		},
		{
			name: "batch-confirm",
			paint: func(c *ui.Context) {
				batchDialog(c, Model{
					ThemeID: "deepseek.light",
					Batch: session.SessionBatch{
						Enabled:    true,
						Confirming: true,
						Kind:       "archive",
						Token:      1,
						Targets:    []session.SessionBatchTarget{{ID: "s1", Title: "Target One"}},
					},
					OnBatchConfirm: func(uint64) {},
					OnBatchCancel:  func(uint64) {},
				})
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tt := ui.NewTester(tc.paint, 800, 600)
			tt.SetTime(time.Now().Add(300 * time.Millisecond))
			tt.Frame()
			boxes := tt.Interactives()
			foundDialog := false
			pointerOrText := 0
			var bad []string
			for _, b := range boxes {
				if b.Role == ui.RoleDialog {
					foundDialog = true
					continue
				}
				// Skip full-window modal backdrop when auditing inner dialog controls.
				if b.Rect.W == 800 && b.Rect.H == 600 {
					continue
				}
				if b.Cursor == ui.CursorPointer || b.Cursor == ui.CursorText {
					pointerOrText++
					continue
				}
				bad = append(bad, b.String())
			}
			if !foundDialog || pointerOrText < 2 {
				t.Fatalf("[%s] foundDialog=%v pointerOrText=%d", tc.name, foundDialog, pointerOrText)
			}
			sort.Strings(bad)
			if len(bad) > 0 {
				t.Fatalf("[%s] dialog controls missing pointer/text cursor:\n%s", tc.name, strings.Join(bad, "\n"))
			}
		})
	}
}

func TestCossSidebarTokensAdaptLightDarkAndGlass(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() {
		i18n.SetLocale(previous)
		design.SetGlass(false, nil)
	})

	for _, glass := range []bool{false, true} {
		design.SetGlass(glass, func(s design.Scheme, p *design.Palette) {
			if s == design.Dark {
				p.Sidebar = ui.RGBA(20, 22, 26, 0.9)
			} else {
				p.Sidebar = ui.RGBA(244, 246, 248, 0.9)
			}
		})
		for _, tc := range []struct {
			name   string
			dark   bool
			scheme design.Scheme
		}{
			{"light", false, design.Light},
			{"dark", true, design.Dark},
		} {
			m := Model{
				SystemDark:      tc.dark,
				ConnectionEpoch: 1,
				Snapshot: session.ViewState{
					WorkspaceReady:  true,
					WorkspaceLayout: session.WorkspaceLayoutState{HostGeneration: 1},
				},
				Batch: session.SessionBatch{
					Enabled:   true,
					Token:     7,
					Kind:      "archive",
					Attempted: 1,
					Results: []session.SessionBatchResult{{
						SessionBatchTarget: session.SessionBatchTarget{ID: "s1", Title: "S1"},
						Error:              "raw-host-internal-error-token",
					}},
				},
				OnBatchEdit: func(string, []session.SessionBatchTarget) {},
			}
			tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 280, 600)
			tt.SetDark(tc.dark)
			wantSidebar := design.Resolve(tc.scheme, ui.Color{}, ui.Color{}).Sidebar
			got := tt.Image().RGBAAt(275, 590)
			if glass && wantSidebar.A < 255 {
				if tc.dark && got.R > 128 || !tc.dark && got.R < 128 {
					t.Fatalf("glass=%v dark=%v unexpected sidebar luminance got=%+v", glass, tc.dark, got)
				}
			} else if got.R != wantSidebar.R || got.G != wantSidebar.G || got.B != wantSidebar.B {
				t.Fatalf("glass=%v dark=%v got=%+v want=%+v", glass, tc.dark, got, wantSidebar)
			}
			if tt.HasText("raw-host-internal-error-token") {
				t.Fatalf("raw host error leaked onto primary sidebar surface: %q", tt.Texts())
			}
		}
	}
}

func TestCossSidebarTimeFormattingUnderUTC(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })

	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	ago := sessionHoverAgo(float64(now.Add(-5*time.Minute).UnixMilli()), now)
	if !strings.Contains(ago, "5m") && !strings.Contains(ago, "5") {
		t.Fatalf("unexpected sessionHoverAgo under UTC: %q", ago)
	}
	created := workspaceCreatedLabel("2026-11-24T12:00:00Z")
	if created == "" {
		t.Fatal("workspaceCreatedLabel returned empty for valid RFC3339 timestamp")
	}
	entry := schedstore.Entry{
		ID:          "t1",
		Title:       "Daily",
		ScheduledAt: now.Add(2 * time.Hour).Format(time.RFC3339),
		Raw:         json.RawMessage(`{"kind":"daily","time":"14:00","timeZone":"UTC"}`),
	}
	abs, rel := schedulePreviewParts(entry, now)
	if abs == "" || !strings.Contains(rel, "2 hour") {
		t.Fatalf("unexpected schedulePreviewParts abs=%q rel=%q", abs, rel)
	}
}
