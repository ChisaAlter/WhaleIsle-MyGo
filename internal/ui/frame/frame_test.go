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
	cols := ComputeColumns(1600, SidebarDefault, DefaultDetailsWidth(1600))
	if cols != (Columns{Sidebar: 280, Center: 600, Details: 720}) {
		t.Fatalf("wide cols=%+v", cols)
	}
	cols = ComputeColumns(1200, SidebarDefault, 540)
	if cols != (Columns{Sidebar: 280, Center: 400, Details: 520}) {
		t.Fatalf("conceded cols=%+v", cols)
	}
	cols = ComputeColumns(900, SidebarDefault, 540)
	if cols != (Columns{Sidebar: 280, Center: 620}) {
		t.Fatalf("auto-closed cols=%+v", cols)
	}
	cols = ComputeColumns(1200, SidebarDefault, 0)
	if cols != (Columns{Sidebar: 280, Center: 920}) {
		t.Fatalf("closed cols=%+v", cols)
	}
}

func TestConversationContentWidthAxis(t *testing.T) {
	// Raw axis remains independent of the rendered seat clearances.
	if got := conversationContentWidth(500); got != 680 {
		t.Fatalf("narrow got %v want 680", got)
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
