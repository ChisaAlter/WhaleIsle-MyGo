package markdown

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestBodyKeyExcludesAvailableWidthAndTracksResolveFile(t *testing.T) {
	base := Style{FontSize: 14, AvailableWidth: 500}
	narrow := base
	narrow.AvailableWidth = 220
	if makeBodyKey(base) != makeBodyKey(narrow) {
		t.Fatal("AvailableWidth change invalidated bodyKey")
	}

	withResolver := base
	withResolver.ResolveFile = func(s string) string { return s }
	if makeBodyKey(withResolver) == makeBodyKey(base) {
		t.Fatal("adding ResolveFile did not change bodyKey")
	}

	// Default ResolveRevision=0 stays stable for unchanged keys.
	sameResolver := withResolver
	sameResolver.ResolveFile = func(s string) string { return "/other/" + s }
	if makeBodyKey(sameResolver) != makeBodyKey(withResolver) {
		t.Fatal("default ResolveRevision=0 unexpectedly changed bodyKey across identical keys")
	}

	rev1 := withResolver
	rev1.ResolveRevision = 1
	rev2 := withResolver
	rev2.ResolveRevision = 2
	if makeBodyKey(rev1) == makeBodyKey(withResolver) || makeBodyKey(rev2) == makeBodyKey(rev1) {
		t.Fatal("incrementing ResolveRevision did not invalidate bodyKey")
	}
}

func TestResizeAvailableWidthKeepsParsedBodyAndUpdatesTableLayout(t *testing.T) {
	src := "See `report.md`.\n\n| ColumnAlpha | ColumnBeta | ColumnGamma | ColumnDelta |\n| --- | --- | --- | --- |\n| value-one-wide-cell | value-two-wide-cell | value-three-wide-cell | value-four-wide-cell |"
	var avail float32 = 640
	var resolveCalls int
	var tableBounds []ui.Rect

	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Children(func() {
			View(c, src, Style{
				FontSize:       14,
				AvailableWidth: avail,
				ResolveFile: func(value string) string {
					resolveCalls++
					return ""
				},
				OnOpenFile: func(string) {},
			})
		}).ObserveLayout(func(r ui.Rect) {
			tableBounds = append(tableBounds, r)
		})
	}, 720, 320)

	if resolveCalls != 1 {
		t.Fatalf("expected 1 initial parse/resolve pass, got %d", resolveCalls)
	}
	wideCell, ok := tt.Find("value-four-wide-cell")
	if !ok {
		t.Fatal("expected wide table cell in initial frame")
	}
	if _, hasCol := tt.Find("ColumnDelta"); !hasCol {
		t.Fatal("expected ColumnDelta visible at wide width")
	}

	// Narrowing AvailableWidth must NOT re-parse AST/spans (resolveCalls stays 1),
	// yet table layout must immediately adapt to the live narrower width.
	avail = 150
	tt.Frame()
	if resolveCalls != 1 {
		t.Fatalf("AvailableWidth resize triggered %d parse/resolve passes; expected cached body (1)", resolveCalls)
	}
	// At width=150 the table wraps/scrolls horizontally inside 150px viewport,
	// so the rightmost cell is clipped off-screen until scrolled horizontally.
	if narrowCell, ok := tt.Find("value-four-wide-cell"); ok && narrowCell.X == wideCell.X && narrowCell.W == wideCell.W {
		t.Fatalf("narrow layout did not adapt table bounds: wide=%+v narrow=%+v", wideCell, narrowCell)
	}
	tt.Scroll(40, wideCell.Y+wideCell.H/2, -260, 0)
	if _, ok := tt.Find("value-four-wide-cell"); !ok {
		t.Fatal("expected horizontally scrolled table to reveal rightmost column in narrow layout")
	}
	if resolveCalls != 1 {
		t.Fatalf("scrolling table triggered re-parse: resolveCalls=%d", resolveCalls)
	}
	_ = tableBounds
}

func TestResolveRevisionInvalidatesCachedFileTargets(t *testing.T) {
	for _, docLayout := range []bool{false, true} {
		name := "default_layout"
		if docLayout {
			name = "document_layout"
		}
		t.Run(name, func(t *testing.T) {
			resolved := false
			var rev uint64
			var opened string
			src := "`report.md`"

			tt := ui.NewTester(func(c *ui.Context) {
				View(c, src, Style{
					FontSize:        14,
					DocumentLayout:  docLayout,
					ResolveRevision: rev,
					ResolveFile: func(value string) string {
						if resolved && value == "report.md" {
							return "/workspace/report.md"
						}
						return ""
					},
					OnOpenFile: func(path string) {
						opened = path
					},
				})
			}, 420, 140)

			box, ok := tt.Find("report.md")
			if !ok {
				t.Fatalf("expected inline code text; texts=%v", tt.Texts())
			}
			tt.ClickAt(box.X+box.W/2, box.Y+box.H/2)
			if opened != "" {
				t.Fatalf("unexpected open before file existed: %q", opened)
			}

			// File becomes resolvable and caller bumps ResolveRevision.
			resolved = true
			rev++
			tt.Frame()

			// The resolved code chip keeps its exact text, so the body's
			// aggregating parent matches Find first; the innermost match is
			// the link element itself.
			matches := tt.FindAll("report.md")
			if len(matches) == 0 {
				t.Fatalf("expected resolved link text; texts=%v", tt.Texts())
			}
			box = matches[len(matches)-1]
			tt.ClickAt(box.X+box.W/2, box.Y+box.H/2)
			if opened != "/workspace/report.md" {
				t.Fatalf("expected resolved file click after ResolveRevision bump, got %q", opened)
			}
		})
	}
}
