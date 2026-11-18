package errcopy

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"whaleisle/internal/ui/i18n"
)

func withLocale(t *testing.T, locale string) {
	t.Helper()
	previous := i18n.Locale()
	i18n.SetLocale(locale)
	t.Cleanup(func() { i18n.SetLocale(previous) })
}

// TestClassifyCoversEveryFamily pins the branch each raw message lands in,
// including the unmatched fallback a person must still get a sentence for.
func TestClassifyCoversEveryFamily(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want Cause
	}{
		{"cancelled by sentinel", context.Canceled, CauseCancelled},
		{"cancelled by text", errors.New("rpc error: code = Canceled desc = context canceled"), CauseCancelled},
		{"deadline by sentinel", context.DeadlineExceeded, CauseTimeout},
		{"timeout by text", errors.New("read tcp 127.0.0.1:1->127.0.0.1:2: i/o timeout"), CauseTimeout},
		{"runtime missing", errors.New("spawn dsh.exe ENOENT"), CauseRuntimeMissing},
		{"port in use", errors.New("listen tcp 127.0.0.1:8787: bind: address already in use"), CausePortInUse},
		{"network", errors.New("dial tcp 127.0.0.1:8080: connect: connection refused"), CauseNetwork},
		{"disk full", errors.New("write /tmp/a: no space left on device"), CauseDiskFull},
		{"read only", errors.New("open /mnt/a: read-only file system"), CauseReadOnly},
		{"permission", errors.New("open C:\\a.txt: Access is denied."), CausePermission},
		{"conflict", errors.New("File changed on disk. Draft kept. Save again to write the draft over the current file."), CauseConflict},
		{"missing", errors.New("open /tmp/none: no such file or directory"), CauseNotFound},
		{"runtime exit", errors.New("dsh web exited: exit status 1"), CauseRuntimeExited},
		{"runtime wrapper", errors.New("dsh web exited"), CauseRuntimeExited},
		{"command exit", errors.New("exit status 128"), CauseExited},
		{"system kill", errors.New("dsh web exited: signal: killed"), CauseKilled},
		{"unmatched", errors.New("revision mismatch"), CauseUnknown},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Classify(tc.err); got != tc.want {
				t.Fatalf("Classify(%q) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// TestSentenceNeverLeaksTheRawText is the contract the shell copy rule depends
// on: the sentence is localized, names the next step, and drops the raw error.
func TestSentenceNeverLeaksTheRawText(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantIn  string
		rawPart string
	}{
		{"runtime missing", errors.New("spawn dsh.exe ENOENT"), "重新安装", "dsh.exe"},
		{"port in use", errors.New("bind: address already in use"), "端口", "address already in use"},
		{"network", errors.New("dial tcp 10.0.0.1:443: connect: connection refused"), "网络", "dial tcp"},
		{"timeout", errors.New("Post https://api: context deadline exceeded"), "超时", "https://api"},
		{"disk full", errors.New("write /tmp/a: no space left on device"), "磁盘空间不足", "/tmp/a"},
		{"read only", errors.New("open /mnt/a: read-only file system"), "不可写入", "/mnt/a"},
		{"permission", errors.New("open C:\\a.txt: Access is denied."), "权限", "C:\\a.txt"},
		{"conflict", errors.New("File changed on disk. Draft kept."), "草稿已保留", "File changed on disk"},
		{"missing", errors.New("open /tmp/none: no such file or directory"), "找不到", "/tmp/none"},
		{"process exit", errors.New("dsh web exited: exit status 1"), "运行时已退出", "exit status"},
		{"unmatched", errors.New("revision mismatch"), "操作没有完成", "revision mismatch"},
	}
	withLocale(t, i18n.LocaleZH)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Sentence(tc.err)
			if !strings.Contains(got, tc.wantIn) {
				t.Fatalf("Sentence(%q) = %q, want it to mention %q", tc.err, got, tc.wantIn)
			}
			if strings.Contains(got, tc.rawPart) {
				t.Fatalf("Sentence(%q) = %q, leaked raw %q", tc.err, got, tc.rawPart)
			}
		})
	}
}

// TestExitCodeNamesTheCodeInPlace covers the startup page's one cause the
// generic exit sentence cannot say: which code the process returned. The code
// is interpolated from the failure's own text and never quoted as the raw
// "exit status N" line.
func TestExitCodeNamesTheCodeInPlace(t *testing.T) {
	withLocale(t, i18n.LocaleZH)
	for _, tc := range []struct{ raw, want string }{
		{"dsh web exited: exit status 1", "桌面运行时已退出（退出码 1），可重试或到日志里查看原因。"},
		{"dsh web exited: exit status 137", "桌面运行时已退出（退出码 137），可重试或到日志里查看原因。"},
	} {
		got := SentenceText(tc.raw)
		if got != tc.want {
			t.Errorf("SentenceText(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
	// No code in the text: the family's own sentence, not a placeholder.
	if got := SentenceText("dsh web exited"); got != i18n.T(keyRuntimeExited) {
		t.Errorf("codeless runtime exit = %q", got)
	}
	// A command that returned non-zero is not the desktop runtime: its status
	// stays in the log, and the sentence must not claim the app exited.
	for _, raw := range []string{"exit status 128", "git: exit status 128"} {
		got := SentenceText(raw)
		if strings.Contains(got, "退出码") || got != i18n.T(keyExited) {
			t.Errorf("command exit %q took the runtime sentence: %q", raw, got)
		}
	}
	if got := SentenceText("open /tmp/a: no such file or directory"); strings.Contains(got, "退出码") {
		t.Errorf("a non-exit failure took the exit sentence: %q", got)
	}
}

// TestSystemKillIsItsOwnSentence keeps a killed process from reading as a
// plain exit: the person's next step differs when the OS stopped it.
func TestSystemKillIsItsOwnSentence(t *testing.T) {
	withLocale(t, i18n.LocaleZH)
	for _, raw := range []string{"dsh web exited: signal: killed", "signal: killed"} {
		got := SentenceText(raw)
		if !strings.Contains(got, "被系统结束") {
			t.Errorf("SentenceText(%q) = %q, want the system-kill sentence", raw, got)
		}
		if strings.Contains(got, "killed") {
			t.Errorf("SentenceText(%q) = %q leaked the raw signal", raw, got)
		}
	}
	if got := Classify(errors.New("dsh web exited: signal: killed")); got != CauseKilled {
		t.Fatalf("Classify(killed) = %v, want CauseKilled", got)
	}
}

func TestSentenceSpeaksTheActiveLocales(t *testing.T) {
	err := errors.New("dial tcp 10.0.0.1:443: connect: connection refused")
	withLocale(t, i18n.LocaleEN)
	if got := Sentence(err); !strings.Contains(got, "network is unavailable") || strings.Contains(got, "dial tcp") {
		t.Fatalf("en sentence = %q", got)
	}
	withLocale(t, i18n.LocaleZH)
	if got := Sentence(err); !strings.Contains(got, "网络连接不可用") {
		t.Fatalf("zh sentence = %q", got)
	}
}

func TestSentenceIsEmptyWithoutText(t *testing.T) {
	if got := Sentence(nil); got != "" {
		t.Fatalf("nil err = %q", got)
	}
	if got := SentenceText("   "); got != "" {
		t.Fatalf("blank text = %q", got)
	}
}

// TestUnderLeadsWithTheAction keeps "what failed" in front of "why", with one
// separator even when the caller's lead already ends in a colon.
func TestUnderLeadsWithTheAction(t *testing.T) {
	withLocale(t, i18n.LocaleZH)
	err := errors.New("open C:\\dst: Access is denied.")
	for _, lead := range []string{"保存标题失败", "保存标题失败：", " 保存标题失败 "} {
		got := Under(lead, err)
		if !strings.HasPrefix(got, "保存标题失败：") {
			t.Fatalf("Under(%q) = %q", lead, got)
		}
		if strings.Count(got, "：") != 1 {
			t.Fatalf("Under(%q) = %q has a doubled separator", lead, got)
		}
		if strings.Contains(got, "Access is denied") {
			t.Fatalf("Under(%q) = %q leaked raw", lead, got)
		}
	}
	if got := Under("保存标题失败", nil); got != "保存标题失败" {
		t.Fatalf("bare lead = %q", got)
	}
}

// TestSentenceTextPassesNoRawCopy covers the OS-shaped messages the queue keeps
// as strings: the sentence replaces them rather than quoting them.
func TestSentenceTextPassesNoRawCopy(t *testing.T) {
	withLocale(t, i18n.LocaleZH)
	failure := &os.PathError{Op: "open", Path: `C:\locked.txt`, Err: errors.New("Access is denied.")}
	raw := failure.Error()
	got := SentenceText(raw)
	if strings.Contains(got, "C:\\locked.txt") || !strings.Contains(got, "权限") {
		t.Fatalf("SentenceText(%q) = %q", raw, got)
	}
}
