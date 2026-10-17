package frame

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestClampSidebarWidth(t *testing.T) {
	if got := clampSidebarWidth(200); got != SidebarMin {
		t.Fatalf("min got %v", got)
	}
	if got := clampSidebarWidth(500); got != SidebarMax {
		t.Fatalf("max got %v", got)
	}
	if got := clampSidebarWidth(300); got != 300 {
		t.Fatalf("mid got %v", got)
	}
}

func TestComputeColumnsDesktopContract(t *testing.T) {
	// Wide: preferred details kept, center absorbs remainder.
	cols := ComputeColumns(1600, SidebarDefault, DetailsDefault)
	if cols.Sidebar != SidebarDefault || cols.Details != DetailsDefault {
		t.Fatalf("wide prefs got %+v", cols)
	}
	if cols.Center != 1600-SidebarDefault-DetailsDefault {
		t.Fatalf("wide center got %v", cols.Center)
	}
	// Mid: shrink details to keep CENTER_MIN.
	cols = ComputeColumns(1220, SidebarDefault, DetailsDefault) // 280+360+640=1280 > 1220
	if cols.Center != CenterMin {
		t.Fatalf("shrink center got %v want %v", cols.Center, CenterMin)
	}
	if cols.Details != 1220-SidebarDefault-CenterMin { // 300
		t.Fatalf("shrink details got %v", cols.Details)
	}
	// Narrow: auto-close details (preference ignored for render).
	cols = ComputeColumns(1200, SidebarDefault, DetailsDefault)
	if cols.Details != 0 {
		t.Fatalf("auto-close details got %v want 0", cols.Details)
	}
	if cols.Center != 1200-SidebarDefault {
		t.Fatalf("auto-close center got %v", cols.Center)
	}
	// Closed details → full center.
	cols = ComputeColumns(1200, SidebarDefault, 0)
	if cols.Details != 0 || cols.Center != 1200-SidebarDefault {
		t.Fatalf("closed got %+v", cols)
	}
}

func TestConversationContentWidthFitsColumn(t *testing.T) {
	// Narrow dialogue column: content must leave 32 DIP for composer card padding.
	if got := conversationContentWidth(500); got != 468 {
		t.Fatalf("narrow got %v want 468", got)
	}
	if got := conversationContentWidth(1200); got != 768 {
		t.Fatalf("wide 64%% got %v want 768", got)
	}
	if got := conversationContentWidth(2000); got != 920 {
		t.Fatalf("cap got %v want 920", got)
	}
}

// TestViewShellLabels renders the M4 main shell headlessly and checks chrome
// that every frame should expose (sidebar, composer, files, welcome line).
func TestViewShellLabels(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)

	st := session.New()
	sizes := DefaultLayout()
	sizes.Bottom = 0 // skip embedded terminal pane in headless tests

	planOpen := true
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{
			Store:         st,
			Sizes:         &sizes,
			ThemeID:       "system",
			SystemDark:    false,
			PlanBoardOpen: &planOpen,
		})
	}, 1200, 800)

	want := []string{
		i18n.T(i18n.KeySidebarNewSessionFallback), // Electron desk-rail primary CTA
		i18n.T(i18n.KeySidebarMarket),
		i18n.T(i18n.KeySidebarSchedule),
		i18n.T(i18n.KeySidebarWorkspaceSection),
		"↑", // circular send control (Electron parity)
		// Empty store → Electron empty-session hero (no 对话/轨迹 chrome).
		i18n.T(i18n.KeyChatHeroSlogan),
		i18n.T(i18n.KeyChatHeroPreview),
	}
	texts := tt.Texts()
	joined := strings.Join(texts, "\n")
	for _, label := range want {
		if tt.HasText(label) {
			continue
		}
		// Primary CTA may be prefixed with a symbol (⊕ / +).
		if strings.Contains(joined, label) {
			continue
		}
		t.Fatalf("missing %q; texts = %q", label, texts)
	}
}
