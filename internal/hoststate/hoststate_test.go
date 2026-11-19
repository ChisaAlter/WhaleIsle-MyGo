package hoststate

import (
	"errors"
	"strings"
	"testing"

	"whaleisle/internal/harness"
	"whaleisle/internal/ui/i18n"
)

// rawTokens are the strings that only ever belong in the log drawer: a page
// that showed one of them would be quoting the transport at a person.
var rawTokens = []string{
	"Host", "host", "dsh", "DSH", "dshVersion", "node.exe", "bin.js", "127.0.0.1",
	"http://", ".json", ".sock", "exit status", "errno", "0.2.",
}

func withLocale(t *testing.T, locale string) {
	t.Helper()
	previous := i18n.Locale()
	i18n.SetLocale(locale)
	t.Cleanup(func() { i18n.SetLocale(previous) })
}

// TestReportWritesEveryStateForAPerson pins what each state says: the sentence
// names what happened, the next step, and nothing a person would have to look
// up. The exact copy is asserted because a change to it is a product decision.
func TestReportWritesEveryStateForAPerson(t *testing.T) {
	withLocale(t, i18n.LocaleZH)
	tests := []struct {
		name        string
		status      harness.Status
		phase       Phase
		line        string
		action      string
		wantInLine  string
		wantAction  bool
		emptyForAll bool
	}{
		{
			name: "never asked to start", status: harness.Status{State: harness.Stopped},
			phase: NotStarted, line: "本地服务没有运行。点击“启动”即可开始使用。", action: "启动", wantInLine: "启动", wantAction: true,
		},
		{
			name: "starting", status: harness.Status{State: harness.Starting},
			phase: Starting, line: "正在启动本地服务，请稍候。", wantInLine: "正在启动",
		},
		{
			name: "ready", status: harness.Status{State: harness.Ready}, phase: Ready, emptyForAll: true,
		},
		{
			name: "failed without a cause", status: harness.Status{State: harness.Failed},
			phase: Failed, line: "本地服务没有启动成功，请重试；原因见日志。", action: "重试", wantInLine: "重试", wantAction: true,
		},
		{
			name: "runtime never came up", status: harness.Status{State: harness.Failed, Err: errors.New("vendored harness not found; set WHALEISLE_VENDOR or run scripts/build-harness.ps1")},
			phase: Failed, line: i18n.T("errcopy.runtime_missing"), action: "重试", wantInLine: "重新安装", wantAction: true,
		},
		{
			name: "runtime is not this build", status: harness.Status{State: harness.Failed, Err: errors.New("compatible Host runtime has unsupported provenance or visual-readiness patch")},
			phase: Failed, line: i18n.T("errcopy.runtime_mismatch"), action: "重试", wantInLine: "不匹配", wantAction: true,
		},
		{
			name: "port range exhausted", status: harness.Status{State: harness.Failed, Err: errors.New("no free port in 3080-3129 on 127.0.0.1")},
			phase: Failed, line: i18n.T("errcopy.port_in_use"), action: "重试", wantInLine: "端口", wantAction: true,
		},
		{
			name: "exited after ready", status: harness.Status{State: harness.Failed, RuntimeFailure: true, Err: errors.New("dsh web exited: exit status 1")},
			phase: Failed, line: "桌面端意外退出，可立即重启；原因见日志。", action: "立即重启", wantInLine: "立即重启", wantAction: true,
		},
		{
			name: "automatic recovery cancelled", status: harness.Status{State: harness.Failed, RecoveryCancelled: true},
			phase: Failed, line: "已取消自动恢复，可手动重试。", action: "重试", wantInLine: "重试", wantAction: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Of(tc.status)
			if got.Phase != tc.phase {
				t.Fatalf("phase = %v, want %v", got.Phase, tc.phase)
			}
			if tc.emptyForAll {
				if got.Line != "" || got.Note != "" || got.Placeholder != "" || got.Action != "" {
					t.Fatalf("ready report carries copy: %+v", got)
				}
				return
			}
			if got.Line != tc.line {
				t.Fatalf("line = %q, want %q", got.Line, tc.line)
			}
			if !strings.Contains(got.Line, tc.wantInLine) {
				t.Fatalf("line %q does not name the next step %q", got.Line, tc.wantInLine)
			}
			if tc.wantAction && got.Action != tc.action {
				t.Fatalf("action = %q, want %q", got.Action, tc.action)
			}
			if !tc.wantAction && got.Action != tc.action {
				t.Fatalf("action = %q, want none", got.Action)
			}
			if got.Note == "" || got.Placeholder == "" {
				t.Fatalf("state %v leaves the sidebar or composer without copy: %+v", got.Phase, got)
			}
		})
	}
}

// TestNoReportQuotesTheTransport keeps the copy rule for every state and both
// locales: no raw technical token, no path, no port.
func TestNoReportQuotesTheTransport(t *testing.T) {
	statuses := []harness.Status{
		{State: harness.Stopped},
		{State: harness.Starting},
		{State: harness.Ready},
		{State: harness.Failed},
		{State: harness.Failed, Err: errors.New("dsh web exited: exit status 3")},
		{State: harness.Failed, Err: errors.New(`spawn C:\apps\node.exe ENOENT`)},
		{State: harness.Failed, Err: errors.New("listen tcp 127.0.0.1:3080: bind: address already in use")},
		{State: harness.Failed, Err: errors.New("compatible Host runtime artifact changed: /r/apps/cli/lib/bin.js")},
		{State: harness.Failed, RuntimeFailure: true, Err: errors.New("dsh web exited: signal: killed")},
		{State: harness.Failed, RecoveryCancelled: true},
	}
	for _, locale := range []string{i18n.LocaleZH, i18n.LocaleEN} {
		withLocale(t, locale)
		for _, st := range statuses {
			report := Of(st)
			for field, line := range map[string]string{"line": report.Line, "note": report.Note, "placeholder": report.Placeholder, "action": report.Action} {
				for _, token := range rawTokens {
					if strings.Contains(line, token) {
						t.Fatalf("%s locale %s quotes %q in its %s: %q", st.State, locale, token, field, line)
					}
				}
			}
		}
	}
}

// TestPhaseOfEverySupervisorState keeps one state from reading as another: the
// surfaces must not call a starting Host failed or a stopped one ready.
func TestPhaseOfEverySupervisorState(t *testing.T) {
	for _, tc := range []struct {
		state harness.State
		want  Phase
	}{
		{harness.Stopped, NotStarted},
		{harness.Starting, Starting},
		{harness.Ready, Ready},
		{harness.Failed, Failed},
	} {
		if got := PhaseOf(harness.Status{State: tc.state}); got != tc.want {
			t.Fatalf("PhaseOf(%v) = %v, want %v", tc.state, got, tc.want)
		}
	}
}
