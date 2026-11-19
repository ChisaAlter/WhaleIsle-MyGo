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
		{"runtime without its bundled node", errors.New("selected Host runtime is missing its bundled Node: /r/node/node"), CauseRuntimeMissing},
		{"harness never staged", errors.New("vendored harness not found; set WHALEISLE_VENDOR or run scripts/build-harness.ps1"), CauseRuntimeMissing},
		{"harness staged but not built", errors.New("vendored harness not built (missing /r/apps/web/dist/index.html); run scripts/build-harness.ps1"), CauseRuntimeMissing},
		{"node absent", errors.New("Node.js 22.19+ is required to run DeepSeek Harness; install it or set nodeBin"), CauseRuntimeMissing},
		{"runtime provenance", errors.New("compatible Host runtime has unsupported provenance or visual-readiness patch"), CauseRuntimeMismatch},
		{"session reader older than the schema", errors.New("Host session reader is older than the shared desktop session schema (missing session.created): /r; stage a complete supported runtime with scripts/session-runtime.mjs; session files were not changed"), CauseRuntimeMismatch},
		{"artifact changed under the app", errors.New("compatible Host runtime artifact changed: /r/apps/cli/lib/bin.js"), CauseRuntimeMismatch},
		{"node paired with the wrong runtime", errors.New("selected Node differs from the verified Host session runtime; use its matching bundled Node"), CauseRuntimeMismatch},
		{"port in use", errors.New("listen tcp 127.0.0.1:8787: bind: address already in use"), CausePortInUse},
		{"port range exhausted", errors.New("no free port in 3080-3129 on 127.0.0.1"), CausePortInUse},
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
		{"provider without a key", errors.New(`llm-deepseek: no API key for provider route "deepseek-official"`), CauseMissingCredential},
		{"provider without a key, other route", errors.New(`llm-openai: no API key for provider route "openai"`), CauseMissingCredential},
		{"bare missing key", errors.New("missing API key"), CauseMissingCredential},
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
		{"runtime without its bundled node", errors.New("selected Host runtime is missing its bundled Node: C:\\r\\node\\node.exe"), "重新安装", "node.exe"},
		{"harness never staged", errors.New("vendored harness not found; set WHALEISLE_VENDOR or run scripts/build-harness.ps1"), "重新安装", "WHALEISLE_VENDOR"},
		{"runtime version mismatch", errors.New("compatible Host runtime has unsupported provenance or visual-readiness patch"), "不匹配", "compatible Host runtime"},
		{"port range exhausted", errors.New("no free port in 3080-3129 on 127.0.0.1"), "端口", "3080"},
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

// TestMissingCredentialIsOneSentenceForEveryRoute pins the family the Host
// reports per provider route: the sentence names the cause and the place to fix
// it, and never the route, the environment variable or the transport's wording.
func TestMissingCredentialIsOneSentenceForEveryRoute(t *testing.T) {
	shapes := []struct{ code, raw string }{
		{"MISSING_CREDENTIAL", `llm-deepseek: no API key for provider route "deepseek-official"; store DEEPSEEK_API_KEY through the credentials service (the web Models page writes it), or export DEEPSEEK_API_KEY in the launching environment`},
		{"", `llm-deepseek: no API key for provider route "deepseek-official"`},
		{"", `llm-openai: no API key for provider route "openai-responses"`},
		{"MISSING_API_KEY", `llm-moonshot: no API key for provider route "moonshot"`},
		{"", "missing api key"},
		{"", "API key is not configured"},
		{"", "未配置 API Key"},
		{"MISSING_CREDENTIAL", ""},
	}
	withLocale(t, i18n.LocaleZH)
	for _, shape := range shapes {
		got := SentenceFor(shape.code, shape.raw)
		if got != i18n.T(keyNoCredential) {
			t.Fatalf("SentenceFor(%q, %q) = %q", shape.code, shape.raw, got)
		}
		if err := ClassifyCode(shape.code, shape.raw); err != CauseMissingCredential {
			t.Fatalf("ClassifyCode(%q, %q) = %v", shape.code, shape.raw, err)
		}
	}
	withLocale(t, i18n.LocaleEN)
	got := SentenceFor("MISSING_CREDENTIAL", `llm-deepseek: no API key for provider route "deepseek-official"`)
	if !strings.Contains(got, "Settings → Models") || strings.Contains(got, "deepseek") {
		t.Fatalf("en sentence = %q", got)
	}
}

// TestProviderEnvelopeNeverReachesCopy is the guard behind the copy rule: any
// failure the Host names after a provider route is copy, whatever the route or
// whether this build knows the cause. A surface can therefore hand the
// classifier a transport string and print the result.
func TestProviderEnvelopeNeverReachesCopy(t *testing.T) {
	envelopes := []string{
		`llm-deepseek: no API key for provider route "deepseek-official"`,
		`llm-openai: no API key for provider route "openai"`,
		`llm-anthropic: upstream 529 overloaded`,
		`llm-google: request rejected: model not available in this region`,
		`llm-deepseek: usage limit reached for this account`,
		`llm-x: unknown provider route`,
	}
	leaks := []string{"llm-", "provider route", "_API_KEY", "API_KEY", "route \""}
	for _, locale := range []string{i18n.LocaleZH, i18n.LocaleEN} {
		withLocale(t, locale)
		for _, raw := range envelopes {
			if !ProviderEnvelope(raw) {
				t.Fatalf("%q was not read as a provider envelope", raw)
			}
			got := SentenceText(raw)
			if got == "" || got == raw {
				t.Fatalf("%s: SentenceText(%q) = %q, want a sentence", locale, raw, got)
			}
			for _, leak := range leaks {
				if strings.Contains(got, leak) {
					t.Fatalf("%s: SentenceText(%q) = %q leaked %q", locale, raw, got, leak)
				}
			}
		}
	}
}

// TestProviderEnvelopeIsNarrow keeps the guard off copy a person wrote or a
// message that only mentions a provider.
func TestProviderEnvelopeIsNarrow(t *testing.T) {
	for _, text := range []string{"会话命名", "Fix the llm-deepseek retry", "llm-deepseek", "DeepSeek 官方", ""} {
		if ProviderEnvelope(text) {
			t.Fatalf("%q read as a provider envelope", text)
		}
	}
}

// TestUnderNamesAnUnknownAction keeps the one case the classifier cannot explain
// from reading as "an operation failed": the lead is the subject, and the
// sentence still ends with the next step.
func TestUnderNamesAnUnknownAction(t *testing.T) {
	for _, locale := range []string{i18n.LocaleZH, i18n.LocaleEN} {
		withLocale(t, locale)
		got := Under("刷新 Agent 目录", errors.New("revision mismatch"))
		if !strings.Contains(got, "刷新 Agent 目录") || strings.Contains(got, "revision mismatch") {
			t.Fatalf("%s: Under(unclassified) = %q", locale, got)
		}
		if strings.Contains(got, "操作") || strings.Contains(got, "operation did not finish") {
			t.Fatalf("%s: Under(unclassified) = %q kept the subjectless sentence", locale, got)
		}
		want := "没有完成。请重试；若仍然失败，请查看日志。"
		if locale == i18n.LocaleEN {
			want = "did not finish. Try again; if it keeps failing, check the log."
		}
		if !strings.HasSuffix(got, want) {
			t.Fatalf("%s: Under(unclassified) = %q, want the tail %q", locale, got, want)
		}
		if got := UnderText("保存设置", "revision mismatch"); !strings.Contains(got, "保存设置") {
			t.Fatalf("%s: UnderText(unclassified) = %q", locale, got)
		}
	}
	withLocale(t, i18n.LocaleZH)
	if got := Under("保存设置", errors.New("dial tcp: connection refused")); !strings.HasPrefix(got, "保存设置：") {
		t.Fatalf("a classified failure keeps the separator: %q", got)
	}
}

// TestHTTPRejectionNamesTheStatusAndTheStep pins the family a Host reports as a
// code: the status is quoted inside the sentence, and a credential rejection or
// a rate limit each says what to do differently.
func TestHTTPRejectionNamesTheStatusAndTheStep(t *testing.T) {
	withLocale(t, i18n.LocaleZH)
	for _, tc := range []struct {
		code, raw, want string
	}{
		{"HTTP_401", "", "检查 API Key"},
		{"", "bad request: HTTP 403", "检查 API Key"},
		{"HTTP_429", "", "限流"},
		{"HTTP_502", "", "模型服务这次出错了"},
		{"HTTP_404", "", "模型名称与地址"},
	} {
		got := SentenceFor(tc.code, tc.raw)
		if !strings.Contains(got, tc.want) {
			t.Errorf("SentenceFor(%q, %q) = %q, want %q", tc.code, tc.raw, got, tc.want)
		}
		if status, ok := HTTPStatus(tc.code, tc.raw); !ok || !strings.Contains(got, status) {
			t.Errorf("SentenceFor(%q, %q) = %q dropped the status", tc.code, tc.raw, got)
		}
		for _, leak := range []string{"HTTP_", "HTTP 4"} {
			if strings.Contains(got, leak) {
				t.Errorf("SentenceFor(%q, %q) = %q leaked %q", tc.code, tc.raw, got, leak)
			}
		}
	}
	if _, ok := HTTPStatus("MISSING_CREDENTIAL", "no api key"); ok {
		t.Fatal("a credential code read as an HTTP status")
	}
}
