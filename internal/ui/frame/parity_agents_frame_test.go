package frame

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestSessionInspectorTableIDHeaderAndLocalesParity(t *testing.T) {
	defer i18n.SetLocale(i18n.Locale())

	snap := session.InspectorSnapshot{
		Binding:   "b1",
		SessionID: "s1",
		Records: []session.InspectorRecord{
			{
				Key:      "r1",
				Identity: "#1",
				Type:     "turn/started",
				Time:     1700000000000,
				Raw:      json.RawMessage(`{"type":"turn/started","time":1700000000000}`),
			},
		},
	}

	for _, tc := range []struct {
		locale   string
		wantCols []string
	}{
		{
			locale:   i18n.LocaleZH,
			wantCols: []string{"序号", "类型", "T/S", "时间 (UTC)", "数据"},
		},
		{
			locale:   i18n.LocaleEN,
			wantCols: []string{"ID", "Type", "T/S", "Time (UTC)", "Data"},
		},
	} {
		t.Run(tc.locale, func(t *testing.T) {
			i18n.SetLocale(tc.locale)
			pane := &SessionInspectorPaneState{Mode: "session-log", selected: "r1"}
			colors := theme.Resolve("system").Active(false)
			tt := ui.NewTester(func(c *ui.Context) {
				inspectorPresentation(c, pane, snap, session.ViewState{}, colors, func() {})
			}, 720, 480)

			texts := strings.Join(tt.Texts(), " | ")
			for _, col := range tc.wantCols {
				if !tt.HasText(col) && !strings.Contains(texts, col) {
					t.Fatalf("locale %s missing column %q in texts=%v", tc.locale, col, tt.Texts())
				}
			}

			for _, it := range tt.Interactives() {
				if !slices.Contains(it.Kinds, "click") || it.Role == ui.RoleSplitter || it.Role == ui.RoleRow || it.Role == ui.RoleCell || it.Role == ui.RoleColumnHeader {
					continue
				}
				if it.Cursor != ui.CursorPointer {
					t.Fatalf("locale %s inspector button %q cursor = %v, want CursorPointer", tc.locale, it.Label, it.Cursor)
				}
			}
		})
	}
}

func TestCordisPanelAndRollbackParity(t *testing.T) {
	defer i18n.SetLocale(i18n.Locale())

	plugins := []CordisPluginRow{
		{
			ID:               "pkg-weather",
			Name:             "Weather Widget",
			Purpose:          "Show live forecast",
			Status:           "running",
			CurrentPackageID: "pkg-weather@1.2.0",
			NextPackageID:    "pkg-weather@1.3.0",
			CurrentSession:   true,
			CanStop:          true,
			CanRollback:      true,
			CanApprove:       true,
		},
		{
			ID:               "pkg-calc",
			Name:             "Calculator",
			Status:           "failed",
			CurrentPackageID: "pkg-calc@0.1.0",
			CurrentSession:   false,
			CanRetry:         true,
			Error:            "syntax error",
		},
	}

	for _, tc := range []struct {
		locale       string
		wantCurrent  string
		wantNext     string
		wantRollback string
		wantGroup1   string
		wantGroup2   string
		wantRunning  string
	}{
		{
			locale:       i18n.LocaleZH,
			wantCurrent:  "当前：pkg-weather@1.2.0",
			wantNext:     "待切换：pkg-weather@1.3.0",
			wantRollback: "回退",
			wantGroup1:   "当前会话",
			wantGroup2:   "其他会话",
			wantRunning:  "1 个运行中",
		},
		{
			locale:       i18n.LocaleEN,
			wantCurrent:  "Current: pkg-weather@1.2.0",
			wantNext:     "Next: pkg-weather@1.3.0",
			wantRollback: "Rollback",
			wantGroup1:   "This session",
			wantGroup2:   "Other sessions",
			wantRunning:  "1 running",
		},
	} {
		t.Run(tc.locale, func(t *testing.T) {
			i18n.SetLocale(tc.locale)
			open := true
			rolledBack := ""
			tt := ui.NewTester(func(c *ui.Context) {
				CordisPanelView(c, CordisPanelModel{
					Open:           &open,
					ThemeID:        "system",
					RetryAvailable: true,
					Plugins:        plugins,
					OnRollback: func(id string) {
						rolledBack = id
					},
				})
				CordisDefineCardView(c, CordisDefineCardModel{
					IsUpdate:   true,
					State:      "defined",
					Name:       "Weather Widget",
					Source:     "export default {}",
					Superseded: true,
				})
			}, 540, 640)

			texts := strings.Join(tt.Texts(), "\n")
			for _, want := range []string{tc.wantCurrent, tc.wantNext, tc.wantRollback, tc.wantGroup1, tc.wantGroup2, tc.wantRunning} {
				if !strings.Contains(texts, want) {
					t.Fatalf("locale %s cordis panel missing %q in:\n%s", tc.locale, want, texts)
				}
			}

			for _, it := range tt.Interactives() {
				if !slices.Contains(it.Kinds, "click") {
					continue
				}
				if it.Cursor != ui.CursorPointer {
					t.Fatalf("locale %s cordis interactive %q cursor = %v, want CursorPointer", tc.locale, it.Label, it.Cursor)
				}
			}
			if err := tt.Click(tc.wantRollback); err != nil {
				t.Fatalf("locale %s click rollback %q failed: %v", tc.locale, tc.wantRollback, err)
			}
			if rolledBack != "pkg-weather" {
				t.Fatalf("locale %s rollback click got %q, want pkg-weather", tc.locale, rolledBack)
			}
		})
	}
}
