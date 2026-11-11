package agents

import (
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestSubagentDurationSegmentsParity(t *testing.T) {
	defer i18n.SetLocale(i18n.Locale())

	const (
		sec   = 1000.0
		minMs = 60 * sec
		hrMs  = 60 * minMs
		dayMs = 24 * hrMs
	)

	cases := []struct {
		name    string
		ms      float64
		wantZH  string
		wantEN  string
		exactZH string
		exactEN string
		titleZH string
		titleEN string
	}{
		{
			name:    "seconds",
			ms:      45 * sec,
			wantZH:  "45秒",
			wantEN:  "45s",
			exactZH: "45秒",
			exactEN: "45s",
			titleZH: "精确用时：45秒",
			titleEN: "Exact duration: 45s",
		},
		{
			name:    "minutes",
			ms:      2*minMs + 5*sec,
			wantZH:  "2分5秒",
			wantEN:  "2m 5s",
			exactZH: "2分5秒",
			exactEN: "2m 5s",
			titleZH: "精确用时：2分5秒",
			titleEN: "Exact duration: 2m 5s",
		},
		{
			name:    "hours",
			ms:      1*hrMs + 2*minMs + 3*sec,
			wantZH:  "1小时2分3秒",
			wantEN:  "1h 2m 3s",
			exactZH: "1小时2分3秒",
			exactEN: "1h 2m 3s",
			titleZH: "精确用时：1小时2分3秒",
			titleEN: "Exact duration: 1h 2m 3s",
		},
		{
			name:    "days_only",
			ms:      2 * dayMs,
			wantZH:  "2天",
			wantEN:  "2d",
			exactZH: "2天0小时0分0秒",
			exactEN: "2d 0h 0m 0s",
			titleZH: "精确用时：2天0小时0分0秒",
			titleEN: "Exact duration: 2d 0h 0m 0s",
		},
		{
			name:    "days_hours",
			ms:      2*dayMs + 5*hrMs + 4*minMs + 3*sec,
			wantZH:  "2天5小时",
			wantEN:  "2d 5h",
			exactZH: "2天5小时4分3秒",
			exactEN: "2d 5h 4m 3s",
			titleZH: "精确用时：2天5小时4分3秒",
			titleEN: "Exact duration: 2d 5h 4m 3s",
		},
		{
			name:    "months_only",
			ms:      60 * dayMs,
			wantZH:  "约2个月",
			wantEN:  "~2mo",
			exactZH: "60天0小时0分0秒",
			exactEN: "60d 0h 0m 0s",
			titleZH: "精确用时：60天0小时0分0秒",
			titleEN: "Exact duration: 60d 0h 0m 0s",
		},
		{
			name:    "months_days",
			ms:      65*dayMs + 6*hrMs,
			wantZH:  "约2个月5天",
			wantEN:  "~2mo 5d",
			exactZH: "65天6小时0分0秒",
			exactEN: "65d 6h 0m 0s",
			titleZH: "精确用时：65天6小时0分0秒",
			titleEN: "Exact duration: 65d 6h 0m 0s",
		},
		{
			name:    "years_only",
			ms:      365 * dayMs,
			wantZH:  "约1年",
			wantEN:  "~1y",
			exactZH: "365天0小时0分0秒",
			exactEN: "365d 0h 0m 0s",
			titleZH: "精确用时：365天0小时0分0秒",
			titleEN: "Exact duration: 365d 0h 0m 0s",
		},
		{
			name:    "years_months",
			ms:      (365 + 60) * dayMs,
			wantZH:  "约1年2个月",
			wantEN:  "~1y 2mo",
			exactZH: "425天0小时0分0秒",
			exactEN: "425d 0h 0m 0s",
			titleZH: "精确用时：425天0小时0分0秒",
			titleEN: "Exact duration: 425d 0h 0m 0s",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			i18n.SetLocale(i18n.LocaleZH)
			if got := FormatSubagentDuration(tc.ms); got != tc.wantZH {
				t.Fatalf("zh duration(%s) = %q, want %q", tc.name, got, tc.wantZH)
			}
			if got := FormatSubagentExactDuration(tc.ms); got != tc.exactZH {
				t.Fatalf("zh exact(%s) = %q, want %q", tc.name, got, tc.exactZH)
			}
			if got := FormatSubagentExactDurationTitle(FormatSubagentExactDuration(tc.ms)); got != tc.titleZH {
				t.Fatalf("zh title(%s) = %q, want %q", tc.name, got, tc.titleZH)
			}

			i18n.SetLocale(i18n.LocaleEN)
			if got := FormatSubagentDuration(tc.ms); got != tc.wantEN {
				t.Fatalf("en duration(%s) = %q, want %q", tc.name, got, tc.wantEN)
			}
			if got := FormatSubagentExactDuration(tc.ms); got != tc.exactEN {
				t.Fatalf("en exact(%s) = %q, want %q", tc.name, got, tc.exactEN)
			}
			if got := FormatSubagentExactDurationTitle(FormatSubagentExactDuration(tc.ms)); got != tc.titleEN {
				t.Fatalf("en title(%s) = %q, want %q", tc.name, got, tc.titleEN)
			}
		})
	}
}

func TestSubagentTokensAndModesParity(t *testing.T) {
	defer i18n.SetLocale(i18n.Locale())

	i18n.SetLocale(i18n.LocaleZH)
	if got := AgentsPanelLocale("mode.external"); got != "外部任务" {
		t.Fatalf("zh mode.external = %q", got)
	}
	if got := SubagentLocale("mode.external"); got != "外部任务" {
		t.Fatalf("zh subagent mode.external = %q", got)
	}
	if got := FormatSubagentTokens(512); got != "512 tok" {
		t.Fatalf("zh FormatSubagentTokens(512) = %q", got)
	}
	if got := FormatSubagentTokens(1500); got != "1.5K tok" {
		t.Fatalf("zh FormatSubagentTokens(1500) = %q", got)
	}
	if got := FormatSubagentTokens(2_000_000); got != "2M tok" {
		t.Fatalf("zh FormatSubagentTokens(2M) = %q", got)
	}
	if got := FormatJobsDurationTitle(true, "12秒"); got != "已运行 12秒" {
		t.Fatalf("zh FormatJobsDurationTitle(live) = %q", got)
	}
	if got := FormatJobsDurationTitle(false, "12秒"); got != "耗时 12秒" {
		t.Fatalf("zh FormatJobsDurationTitle(done) = %q", got)
	}
	if got := FormatJobsTerminalExpand(15); got != "展开其余 15 行" {
		t.Fatalf("zh FormatJobsTerminalExpand = %q", got)
	}
	if got := FormatJobsTerminalExitCode(1); got != "退出码 1" {
		t.Fatalf("zh FormatJobsTerminalExitCode = %q", got)
	}
	if got := FormatAgentTeamFailure("bad json"); got != "团队持久记录无效：bad json" {
		t.Fatalf("zh FormatAgentTeamFailure = %q", got)
	}

	i18n.SetLocale(i18n.LocaleEN)
	if got := AgentsPanelLocale("mode.external"); got != "External task" {
		t.Fatalf("en mode.external = %q", got)
	}
	if got := SubagentLocale("mode.external"); got != "External task" {
		t.Fatalf("en subagent mode.external = %q", got)
	}
	if got := FormatJobsDurationTitle(true, "12s"); got != "Running for 12s" {
		t.Fatalf("en FormatJobsDurationTitle(live) = %q", got)
	}
	if got := FormatJobsDurationTitle(false, "12s"); got != "Took 12s" {
		t.Fatalf("en FormatJobsDurationTitle(done) = %q", got)
	}
	if got := FormatJobsTerminalExpand(15); got != "Show 15 more lines" {
		t.Fatalf("en FormatJobsTerminalExpand = %q", got)
	}
	if got := FormatJobsTerminalExitCode(1); got != "exit 1" {
		t.Fatalf("en FormatJobsTerminalExitCode = %q", got)
	}
	if got := FormatAgentTeamFailure("bad json"); got != "Invalid team persistence record: bad json" {
		t.Fatalf("en FormatAgentTeamFailure = %q", got)
	}
}

func TestCatalogRowsExternalAndDurationHeadlessAndCursor(t *testing.T) {
	defer i18n.SetLocale(i18n.Locale())

	tok := 2400.0
	navView := session.AgentNavigationView{
		SessionID: "root",
		Token:     1,
		Rows: []session.AgentNavigationRow{
			{
				Label:         "Coder",
				ActivityKnown: true,
				Running:       true,
				Address:       session.AgentSessionAddress{Kind: "subagent", ParentSessionID: "root", ChildSessionID: "c1", Mode: "continuable"},
				Metrics: session.AgentMetrics{
					Tokens: &tok,
					Timing: &session.AgentTiming{SettledMs: float64((2*24 + 5) * 3600 * 1000)},
				},
			},
			{
				Label:         "Auditor",
				ActivityKnown: true,
				Metrics: session.AgentMetrics{
					Completed: true,
					Timing:    &session.AgentTiming{SettledMs: float64((1*3600 + 2*60 + 3) * 1000)},
				},
				Address: session.AgentSessionAddress{Kind: "subagent", ParentSessionID: "root", ChildSessionID: "c2", Mode: "one-shot"},
			},
			{
				Label:   "CI Hook",
				Address: session.AgentSessionAddress{Kind: "subagent", ParentSessionID: "root", ChildSessionID: "ext1", Mode: "external"},
			},
		},
	}

	for _, locale := range []struct {
		loc   string
		wants []string
	}{
		{
			loc:   i18n.LocaleZH,
			wants: []string{"Coder", "2.4K tok", "2天5小时", "正在运行 · 可继续", "Auditor", "1小时2分3秒", "已完成 · 一次性", "CI Hook", "外部任务", "在侧边栏打开"},
		},
		{
			loc:   i18n.LocaleEN,
			wants: []string{"Coder", "2.4K tok", "2d 5h", "running · continuable", "Auditor", "1h 2m 3s", "completed · one-shot", "CI Hook", "External task", "Open in sidebar"},
		},
	} {
		i18n.SetLocale(locale.loc)
		tt := ui.NewTester(func(c *ui.Context) {
			catalogRowsView(c, Model{
				ThemeID:            "system",
				OnOpenConversation: func(_ uint64, _ session.AgentSessionAddress) {},
				OnOpenChat:         func(_ uint64, _ session.AgentSessionAddress) {},
			}, navView, ui.RGB(120, 120, 120))
		}, 520, 400)

		joined := strings.Join(tt.Texts(), " | ")
		for _, want := range locale.wants {
			if !tt.HasText(want) && !strings.Contains(joined, want) {
				t.Fatalf("locale %s missing %q in texts=%v", locale.loc, want, tt.Texts())
			}
		}

		// Verify all clickable buttons in catalogRowsView have CursorPointer.
		pointerCount := 0
		for _, it := range tt.Interactives() {
			if !slices.Contains(it.Kinds, "click") {
				continue
			}
			// CI Hook is an external task and is disabled (CanOpen() == false).
			if it.Label == "CI Hook" {
				continue
			}
			if it.Cursor != ui.CursorPointer {
				t.Fatalf("locale %s interactive %q cursor = %v, want CursorPointer", locale.loc, it.Label, it.Cursor)
			}
			pointerCount++
		}
		if pointerCount < 4 {
			t.Fatalf("locale %s expected at least 4 pointer buttons, got %d", locale.loc, pointerCount)
		}
	}
}
