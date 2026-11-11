package settings

import (
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/shortcuts"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func assertInteractiveCursors(t *testing.T, section string, tt *ui.Tester) {
	t.Helper()
	boxes := tt.Interactives()
	for _, b := range boxes {
		if slices.Contains(b.Kinds, "text") {
			if b.Cursor != ui.CursorText && b.Cursor != ui.CursorDefault && b.Cursor != ui.CursorNotAllowed {
				t.Errorf("[%s] text input %s has unexpected cursor %v", section, b.String(), b.Cursor)
			}
			continue
		}
		if slices.Contains(b.Kinds, "click") {
			if b.Cursor != ui.CursorPointer && b.Cursor != ui.CursorDefault && b.Cursor != ui.CursorNotAllowed {
				t.Errorf("[%s] clickable %s has unexpected cursor %v", section, b.String(), b.Cursor)
			}
		}
	}
}

// TestCursorAuditSettings renders the settings page headlessly across all
// categories and reports interactive element cursors.
func TestCursorAuditSettings(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)

	tr := true
	fl := false
	loc := "zh"
	selectedModel := "deepseek-chat"
	themeMode := "system"

	sections := []struct {
		name  string
		model Model
	}{
		{
			name:  "general",
			model: Model{DesktopActive: true, Desktop: &DesktopAccess{Preferences: &theme.Editor{}, Shortcuts: &shortcuts.Editor{}, AutoStartDesktop: &tr, Fields: DesktopFields{Locale: &loc, CloseToTray: &tr, OpenAtLogin: &fl}}},
		},
		{
			name:  "appearance",
			model: Model{AppearanceActive: true, Appearance: &AppearanceAccess{Preferences: &theme.Editor{}, Background: &WallpaperEditor{LowPowerGPU: true}, Theme: &themeMode, LowPowerGPU: true}},
		},
		{
			name:  "remote",
			model: Model{RemoteActive: true, Remote: &RemoteAccess{Enabled: true, RelayConnecting: true, PairingRefreshSeconds: 180, PairingURL: "https://whaleisle.example/pair/abc", PersistentLink: true, OnRevokeLink: func() {}, OnRefreshPairing: func() {}, OnCopyLink: func() {}}},
		},
		{
			name:  "models",
			model: Model{ModelsActive: true, Models: &ModelsAccess{Labels: []string{"deepseek-chat", "deepseek-reasoner"}, Selected: &selectedModel}},
		},
		{
			name:  "conversation",
			model: Model{ConversationActive: true, Conversation: &ConversationAccess{}},
		},
		{
			name:  "interface",
			model: Model{InterfaceActive: true, Interface: &InterfaceAccess{Locale: &loc, CloseToTray: &tr, OpenAtLogin: &fl, ShowViewTabs: &tr, ShowArchivedList: &fl, AutoAllowTools: &fl}},
		},
		{
			name:  "features",
			model: Model{FeaturesActive: true, Features: &FeaturesAccess{AssistantEnabled: &tr, RemoteWorkspaceEnabled: &fl}},
		},
		{
			name:  "plugins",
			model: Model{PluginsActive: true, Plugins: &PluginsAccess{ClientReady: true, HasListRPC: true, Rows: []PluginRow{{Name: "sample-plugin", Status: "active", CanToggle: true}}}},
		},
		{
			name:  "skills",
			model: Model{SkillsActive: true, Skills: &SkillsAccess{SessionID: "sess-1", Skills: []SkillRow{{Name: "sample-skill", Description: "desc", Writable: true, UserInvocable: true}}}},
		},
		{
			name:  "mcp",
			model: Model{MCPActive: true, MCP: &MCPAccess{HasListRPC: true, Servers: []MCPServerRow{{ServerName: "local-mcp", ID: "mcp-1", Enabled: true, Writable: true, Status: "connected"}}}},
		},
		{
			name:  "agent-presets",
			model: Model{AgentPresetsActive: true, AgentPresets: &AgentPresetsAccess{Selected: &selectedModel}},
		},
		{
			name:  "account",
			model: Model{AccountActive: true, Account: &AccountAccess{ClientReady: true, Status: "signed_in", Email: "user@example.com", Name: "Tester"}},
		},
		{
			name:  "about-paused",
			model: Model{AboutActive: true, About: &AboutAccess{AppName: "Whale Isle", Version: "1.0.0", UpdatePhase: "paused", UpdateVersion: "1.1.0", UpdatePercent: 45, OnResumeUpdate: func() {}, OnCheckUpdates: func() {}, OnOpenUpdateCenter: func() {}}},
		},
		{
			name:  "about-ready",
			model: Model{AboutActive: true, About: &AboutAccess{AppName: "Whale Isle", Version: "1.0.0", UpdatePhase: "ready", UpdateVersion: "1.1.0", OnInstallUpdate: func() {}, OnOpenUpdateCenter: func() {}}},
		},
		{
			name:  "about-error",
			model: Model{AboutActive: true, About: &AboutAccess{AppName: "Whale Isle", Version: "1.0.0", UpdatePhase: "error", Err: "network timeout", OnCheckUpdates: func() {}, OnOpenUpdateCenter: func() {}}},
		},
	}

	for _, tc := range sections {
		t.Run(tc.name, func(t *testing.T) {
			m := tc.model
			if m.Desktop == nil {
				m.Desktop = &DesktopAccess{}
			}
			tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { View(c, m) }) }, 1000, 900)
			tt.Frame()
			assertInteractiveCursors(t, tc.name, tt)
			boxes := tt.Interactives()
			var lines []string
			for _, b := range boxes {
				if b.Cursor == ui.CursorPointer {
					continue
				}
				lines = append(lines, b.String())
			}
			sort.Strings(lines)
			t.Logf("[%s] %d interactive elements, %d without a hand:\n%s", tc.name, len(boxes), len(lines), strings.Join(lines, "\n"))
		})
	}
}
