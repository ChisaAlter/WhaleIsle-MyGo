package chat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func contextBodyFixture(source, content string) session.Message {
	return session.Message{
		ID: "context-body-fixture", Kind: session.MessageKindContext, Role: "context", Seq: 7,
		Body: "LOSSY PROJECTION MUST NOT APPEAR",
		Raw:  json.RawMessage(`{"type":"user/message","data":{"source":` + source + `,"content":` + content + `}}`),
	}
}

func contextBodyTestStyle() contextBodyStyle {
	return contextBodyStyle{
		Font: "monospace", Size: 11, LineHeight: 16,
		Secondary: ui.RGB(40, 40, 40), Tertiary: ui.RGB(70, 70, 70),
		Caption: ui.RGB(90, 90, 90), Border: ui.RGB(180, 180, 180),
	}
}

func contextBodyTester(body contextBody) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) { body.paint(c, contextBodyTestStyle()) }, 1000, 1800)
}

func contextBodyEnglish(t *testing.T) {
	t.Helper()
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(old) })
}

func contextBodyExpectText(t *testing.T, tt *ui.Tester, text string) ui.Rect {
	t.Helper()
	r, ok := tt.Find(text)
	if !ok {
		t.Fatalf("missing %q; visible text: %q", text, tt.Texts())
	}
	return r
}

func TestContextBodyOrderedRunsRetainUnknownJSONInPlace(t *testing.T) {
	contextBodyEnglish(t)
	unknown := `{ "type":"foreign", "extra":{"z":9007199254740993,"a":[true,null]}, "untouched":"\\u0061" }`
	msg := contextBodyFixture(`{"kind":"foreign","form":"future","tail":"value"}`, `[{"type":"text","text":"first"},{"type":"text","text":" joined"},`+unknown+`,{"type":"text","text":"last\n  exact  "},{"type":"text","text":""}]`)
	original := append([]byte(nil), msg.Raw...)
	body := resolveContextBody(msg)
	if body.Form != "" || body.Summary != "" {
		t.Fatalf("unknown declaration resolved as %q / %q", body.Form, body.Summary)
	}
	tt := contextBodyTester(body)
	first := contextBodyExpectText(t, tt, "first joined")
	block := contextBodyExpectText(t, tt, "Unknown content block")
	last := contextBodyExpectText(t, tt, "last\n  exact  ")
	if first.Y >= block.Y || block.Y >= last.Y {
		t.Fatal("unknown content moved past adjacent model-facing text")
	}
	if tt.HasText("9007199254740993") || tt.HasText("LOSSY PROJECTION") {
		t.Fatal("closed JSON disclosure leaked its body or used the flattened projection")
	}
	if err := tt.Click("Unknown content block"); err != nil {
		t.Fatal(err)
	}
	contextBodyExpectText(t, tt, contextFormattedJSON(json.RawMessage(unknown)))
	if tt.HasText("Copy JSON") {
		t.Fatal("JSON disclosure introduced an absent reference action")
	}
	if err := tt.Click("Unknown content block"); err != nil {
		t.Fatal(err)
	}
	if _, found := tt.Find("Copy JSON"); found || tt.HasText("9007199254740992") {
		t.Fatal("closing disclosure left its controls or body visible")
	}
	if !bytes.Equal(msg.Raw, original) {
		t.Fatal("resolve or paint mutated the durable raw event")
	}
}

func TestContextBodyJSONMatchesJavaScriptFormatting(t *testing.T) {
	raw := json.RawMessage(`{"z":9007199254740993,"2":"two","1":"one","decoded":"\u0061<>&","negzero":-0,"overflow":1e400,"z":9007199254740993}`)
	want := "{\n  \"1\": \"one\",\n  \"2\": \"two\",\n  \"z\": 9007199254740992,\n  \"decoded\": \"a<>&\",\n  \"negzero\": 0,\n  \"overflow\": null\n}"
	if got := contextFormattedJSON(raw); got != want {
		t.Fatalf("JSON preview=%q want %q", got, want)
	}
	if got := contextJSONStringLiteral(`literal \u2028 and ` + "\u2028"); got != `"literal \\u2028 and `+"\u2028"+`"` {
		t.Fatalf("literal slash or separator changed: %q", got)
	}
}

func TestContextBodyMetadataWhitespaceAndAdjacentJSONSpacing(t *testing.T) {
	contextBodyEnglish(t)
	msg := contextBodyFixture(`{"kind":"foreign","note":"one\n  two\tthree","nbsp":"a\u00a0b"}`, `[{"type":"text","text":"model\n  exact"},{"type":"future","id":1},{"type":"future","id":2}]`)
	tt := contextBodyTester(resolveContextBody(msg))
	contextBodyExpectText(t, tt, "model\n  exact")
	contextBodyExpectText(t, tt, "one two three")
	contextBodyExpectText(t, tt, "a\u00a0b")
	blockCount := 0
	for _, text := range tt.Texts() {
		if text == "Unknown content block" {
			blockCount++
		}
	}
	if blockCount != 2 {
		t.Fatalf("adjacent JSON lost: %q", tt.Texts())
	}
	// The labels share a name; use first/last model runs to locate their combined height.
	msg = contextBodyFixture(`{"kind":"foreign"}`, `[{"type":"future","id":1},{"type":"future","id":2},{"type":"text","text":"after blocks"}]`)
	tt = contextBodyTester(resolveContextBody(msg))
	after := contextBodyExpectText(t, tt, "after blocks")
	if after.Y != 56 {
		t.Fatalf("adjacent JSON margins did not collapse: %+v", after)
	}
}

func TestContextBodySourceFieldsFollowObjectEntriesOrder(t *testing.T) {
	contextBodyEnglish(t)
	msg := contextBodyFixture(`{"kind":"foreign","zeta":"first","10":"ten","form":"future","2":"two","alpha":{"z":1, "a":[true,false]},"01":"leading zero","zeta":"last value","4294967295":null}`, `[{"type":"text","text":"model content"}]`)
	tt := contextBodyTester(resolveContextBody(msg))
	keys := []string{"2", "10", "zeta", "form", "alpha", "01", "4294967295"}
	previous := float32(-1)
	for _, key := range keys {
		box := contextBodyExpectText(t, tt, key)
		if box.Y <= previous {
			t.Fatalf("field %q is not in Object.entries order", key)
		}
		previous = box.Y
	}
	for _, text := range []string{"two", "ten", "last value", "future", `{"z":1,"a":[true,false]}`, "leading zero", "null"} {
		contextBodyExpectText(t, tt, text)
	}
	if _, found := tt.Find("kind"); found {
		t.Fatal("opaque source repeated the producer kind")
	}
	if _, found := tt.Find("first"); found {
		t.Fatal("duplicate property retained its earlier value")
	}
}

func TestContextBoundedTextUsesJavaScriptUTF16Units(t *testing.T) {
	contextBodyEnglish(t)
	cases := []struct {
		name, input, prefix string
		total               int
	}{
		{"ascii exact", strings.Repeat("a", 20000), strings.Repeat("a", 20000), 20000},
		{"ascii over", strings.Repeat("a", 20001), strings.Repeat("a", 20000), 20001},
		{"multibyte exact", strings.Repeat("界", 20000), strings.Repeat("界", 20000), 20000},
		{"multibyte over", strings.Repeat("界", 20001), strings.Repeat("界", 20000), 20001},
		{"emoji exact", strings.Repeat("😀", 10000), strings.Repeat("😀", 10000), 20000},
		{"emoji over", strings.Repeat("😀", 10001), strings.Repeat("😀", 10000), 20002},
		{"surrogate split", strings.Repeat("a", 19999) + "😀tail", strings.Repeat("a", 19999) + "\ufffd", 20005},
		{"emoji after limit", strings.Repeat("界", 20000) + "😀", strings.Repeat("界", 20000), 20002},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := contextBoundedText(tc.input)
			want := tc.prefix
			if tc.total > 20000 {
				want += fmt.Sprintf("\n… truncated, %d characters total", tc.total)
			}
			if got != want {
				t.Errorf("wrong UTF16 bound: got %d UTF8 bytes, want %d", len(got), len(want))
			}
			if !utf8.ValidString(got) {
				t.Fatal("native text contains an invalid UTF8 surrogate fragment")
			}
			if gotPrefix, _, _ := strings.Cut(got, "\n"); len(utf16.Encode([]rune(gotPrefix))) != 20000 {
				t.Fatal("display prefix is not exactly 20,000 JS code units")
			}
		})
	}
	i18n.SetLocale(i18n.LocaleZH)
	if got := contextBoundedText(strings.Repeat("界", 20001)); !strings.HasSuffix(got, "\n… 已截断，共 20001 字符") {
		t.Fatal("truncation footer did not use reference Chinese copy")
	}
}

func TestContextBodyBoundIsPerJoinedRunAndSourceField(t *testing.T) {
	contextBodyEnglish(t)
	long := strings.Repeat("界", 20001)
	quoted, _ := json.Marshal(long)
	msg := contextBodyFixture(`{"kind":"foreign","long":`+string(quoted)+`}`, `[{"type":"text","text":`+string(quoted)+`},{"type":"text","text":"😀"},{"type":"foreign"},{"type":"text","text":"later run"}]`)
	tt := contextBodyTester(resolveContextBody(msg))
	contextBodyExpectText(t, tt, strings.Repeat("界", 20000)+"\n… truncated, 20003 characters total")
	contextBodyExpectText(t, tt, strings.Repeat("界", 20000)+" … truncated, 20001 characters total")
	contextBodyExpectText(t, tt, "later run")
	if tt.HasText("LOSSY PROJECTION") {
		t.Fatal("raw content was replaced with the projected body")
	}
}

func TestContextBodyJSONDisplayBoundKeepsRetainedEvent(t *testing.T) {
	contextBodyEnglish(t)
	payload := `{ "type":"foreign", "value":"` + strings.Repeat("😀", 11000) + `TAIL" }`
	msg := contextBodyFixture(`{"kind":"foreign"}`, `[`+payload+`]`)
	original := append([]byte(nil), msg.Raw...)
	tt := contextBodyTester(resolveContextBody(msg))
	if err := tt.Click("Unknown content block"); err != nil {
		t.Fatal(err)
	}
	formatted := contextFormattedJSON(json.RawMessage(payload))
	total := len(utf16.Encode([]rune(formatted)))
	if !tt.HasText(fmt.Sprintf("… truncated, %d characters total", total)) || tt.HasText("TAIL") {
		t.Fatal("expanded JSON did not apply the reference 20k display bound")
	}
	if tt.HasText("Copy JSON") {
		t.Fatal("JSON display added a reference-absent action")
	}
	if !bytes.Equal(msg.Raw, original) {
		t.Fatal("JSON truncation changed the retained event")
	}
}

func TestContextBodyInstructionsKeepChangesAndFraming(t *testing.T) {
	contextBodyEnglish(t)
	for _, baseline := range []bool{false, true} {
		t.Run(fmt.Sprint(baseline), func(t *testing.T) {
			source := fmt.Sprintf(`{"kind":"agent-instructions","form":"instructions","baseline":%t,"changes":[{"path":"AGENTS.md","action":"set","digest":"sha256:one"},{"path":"AGENTS.md","action":"replace"},{"path":"rules.md","action":"replace","digest":5},{"path":"retired.md","action":"remove"}],"foreign":"hidden metadata"}`, baseline)
			body := resolveContextBody(contextBodyFixture(source, `[{"type":"text","text":"<system-reminder>\n  exact instructions\n</system-reminder>"},{"type":"foreign","retained":true}]`))
			if body.Form != "instructions" || len(body.instructions) != 3 || body.instructions[0].Digest != "sha256:one" {
				t.Fatalf("instructions did not keep first-seen paths: %+v", body)
			}
			tt := contextBodyTester(body)
			for _, text := range []string{"AGENTS.md", "rules.md", "retired.md", "removed", "<system-reminder>\n  exact instructions\n</system-reminder>", "Unknown content block"} {
				contextBodyExpectText(t, tt, text)
			}
			if baseline {
				contextBodyExpectText(t, tt, "loaded")
			} else {
				contextBodyExpectText(t, tt, "added")
				contextBodyExpectText(t, tt, "updated")
			}
			if tt.HasText("hidden metadata") || tt.HasText("baseline") {
				t.Fatal("dedicated instructions repeated source fields")
			}
		})
	}
}

func TestContextBodyCatalogCapsRowsButValidatesEveryEntry(t *testing.T) {
	contextBodyEnglish(t)
	var entries []string
	for i := 0; i < 202; i++ {
		entries = append(entries, fmt.Sprintf(`{"name":"skill-%03d","description":"description-%03d"}`, i, i))
	}
	content := `[{"type":"text","text":"catalog model prose hidden"},{"type":"foreign","retained":"catalog block"}]`
	source := `{"kind":"skills","form":"catalog","update":true,"entries":[` + strings.Join(entries, ",") + `]}`
	body := resolveContextBody(contextBodyFixture(source, content))
	if body.Form != "catalog" || len(body.catalog) != 200 || body.catalogTotal != 202 {
		t.Fatalf("catalog materialized %d of %d entries as %q", len(body.catalog), body.catalogTotal, body.Form)
	}
	tt := contextBodyTester(body)
	for _, text := range []string{"Replacement catalog", "skill-000", "skill-199", "… 2 more", "Unknown content block"} {
		contextBodyExpectText(t, tt, text)
	}
	if tt.HasText("skill-200") || tt.HasText("catalog model prose hidden") {
		t.Fatal("catalog exceeded 200 rows or repeated replaced model prose")
	}
	entries[201] = `{"name":"unreadable last entry"}`
	malformed := resolveContextBody(contextBodyFixture(`{"kind":"skills","form":"catalog","entries":[`+strings.Join(entries, ",")+`]}`, content))
	if malformed.Form != "" {
		t.Fatal("catalog silently ignored an unreadable entry beyond its display cap")
	}
	fallback := contextBodyTester(malformed)
	contextBodyExpectText(t, fallback, "catalog model prose hidden")
	contextBodyExpectText(t, fallback, "form")
}

func TestContextBodyEmptyCatalogRetiresEarlierEntries(t *testing.T) {
	contextBodyEnglish(t)
	msg := contextBodyFixture(`{"kind":"skills","form":"catalog","update":true,"entries":[]}`, `[{"type":"text","text":"replacement prose"},{"type":"foreign"}]`)
	body := resolveContextBody(msg)
	if body.Form != "catalog" || body.catalogTotal != 0 || len(body.catalog) != 0 {
		t.Fatal("empty replacement catalog fell back to opaque")
	}
	tt := contextBodyTester(body)
	contextBodyExpectText(t, tt, "Replacement catalog")
	contextBodyExpectText(t, tt, "Unknown content block")
	if tt.HasText("replacement prose") || tt.HasText("more") {
		t.Fatal("empty catalog invented rows or exposed replaced prose")
	}
	duplicates := resolveContextBody(contextBodyFixture(`{"form":"catalog","entries":[{"name":"same","description":"first"},{"name":"same","description":"second"}]}`, `[]`))
	tt = contextBodyTester(duplicates)
	contextBodyExpectText(t, tt, "first")
	contextBodyExpectText(t, tt, "second")
}

func TestContextBodySnapshotReplacesAllContentWithSections(t *testing.T) {
	contextBodyEnglish(t)
	long := strings.Repeat("界", 20001)
	quoted, _ := json.Marshal(long)
	body := resolveContextBody(contextBodyFixture(`{"kind":"runtime","form":"snapshot","sections":[{"name":"Workspace","text":"  root\nline"},{"name":"Runtime","text":`+string(quoted)+`}]}`, `[{"type":"text","text":"snapshot joined prose"},{"type":"foreign","dropped":"as reference"}]`))
	if body.Form != "snapshot" {
		t.Fatal("readable snapshot did not specialize")
	}
	tt := contextBodyTester(body)
	caption := contextBodyExpectText(t, tt, "Supersedes earlier snapshots")
	first := contextBodyExpectText(t, tt, "Workspace")
	text := contextBodyExpectText(t, tt, "  root\nline")
	second := contextBodyExpectText(t, tt, "Runtime")
	contextBodyExpectText(t, tt, strings.Repeat("界", 20000)+"\n… truncated, 20001 characters total")
	if caption.Y >= first.Y || first.Y >= text.Y || text.Y >= second.Y {
		t.Fatal("snapshot section order or framing caption changed")
	}
	if tt.HasText("snapshot joined prose") || tt.HasText("Unknown content block") {
		t.Fatal("snapshot retained content that reference replaces with sections")
	}
}

func TestContextBodyNoticeRelayAndRecallPresentRecordedFacts(t *testing.T) {
	contextBodyEnglish(t)
	content := `[{"type":"text","text":"first"},{"type":"text","text":" second"},{"type":"foreign"},{"type":"text","text":"last"}]`
	for _, tc := range []struct {
		form, source, summary string
		labels                []string
	}{
		{"notice", `{"kind":"producer","form":"notice","summary":"one line","extra":"hidden metadata"}`, "one line", nil},
		{"relay", `{"kind":"agent-message","form":"relay","senderSessionId":"opaque-agent-id","extra":"hidden metadata"}`, "", []string{"From session opaque-agent-id"}},
		{"recall", `{"kind":"session-reference","form":"recall","references":[{"label":"Source A","retainedMessages":12,"omittedMessages":3,"truncated":true},{"label":"Source B","retainedMessages":-1.5,"omittedMessages":0,"truncated":false}],"extra":"hidden metadata"}`, "", []string{"Source A", "12 kept · 3 omitted", "truncated", "Source B", "-1.5 kept · 0 omitted"}},
	} {
		t.Run(tc.form, func(t *testing.T) {
			body := resolveContextBody(contextBodyFixture(tc.source, content))
			if body.Form != tc.form || body.Summary != tc.summary {
				t.Fatalf("resolved %q / %q, want %q / %q", body.Form, body.Summary, tc.form, tc.summary)
			}
			tt := contextBodyTester(body)
			for _, text := range append(tc.labels, "first second", "Unknown content block", "last") {
				contextBodyExpectText(t, tt, text)
			}
			if tt.HasText("hidden metadata") || tt.HasText("one line") || tt.HasText("senderSessionId") {
				t.Fatal("dedicated body repeated source metadata or collapsed summary")
			}
		})
	}
}

func TestContextBodyMalformedFormsFallBackWithoutLosingFields(t *testing.T) {
	contextBodyEnglish(t)
	cases := []struct{ name, source string }{
		{"unknown", `{"form":"future","foreign":[1,2]}`},
		{"form wrong type", `{"form":7,"foreign":true}`},
		{"instructions absent", `{"form":"instructions"}`},
		{"instructions null", `{"form":"instructions","changes":null}`},
		{"instructions primitive entry", `{"form":"instructions","changes":[true]}`},
		{"instructions empty", `{"form":"instructions","changes":[]}`},
		{"instructions invalid action", `{"form":"instructions","changes":[{"path":"a","action":"future"}]}`},
		{"instructions empty path", `{"form":"instructions","changes":[{"path":"","action":"set"}]}`},
		{"instructions invalid duplicate", `{"form":"instructions","changes":[{"path":"a","action":"set"},{"path":"a","action":"future"}]}`},
		{"catalog absent", `{"form":"catalog"}`},
		{"catalog primitive entry", `{"form":"catalog","entries":[true]}`},
		{"catalog null", `{"form":"catalog","entries":null}`},
		{"catalog partial", `{"form":"catalog","entries":[{"name":"readable","description":"yes"},{"name":"bad","description":null}]}`},
		{"catalog empty name", `{"form":"catalog","entries":[{"name":"","description":"yes"}]}`},
		{"snapshot empty", `{"form":"snapshot","sections":[]}`},
		{"snapshot absent", `{"form":"snapshot"}`},
		{"snapshot primitive entry", `{"form":"snapshot","sections":[true]}`},
		{"snapshot empty name", `{"form":"snapshot","sections":[{"name":"","text":"yes"}]}`},
		{"snapshot unreadable", `{"form":"snapshot","sections":[{"name":"runtime","text":null}]}`},
		{"notice absent summary", `{"form":"notice"}`},
		{"notice empty summary", `{"form":"notice","summary":""}`},
		{"notice invalid summary", `{"form":"notice","summary":true}`},
		{"relay absent sender", `{"form":"relay"}`},
		{"relay empty sender", `{"form":"relay","senderSessionId":""}`},
		{"relay invalid sender", `{"form":"relay","senderSessionId":7}`},
		{"recall empty", `{"form":"recall","references":[]}`},
		{"recall absent", `{"form":"recall"}`},
		{"recall empty label", `{"form":"recall","references":[{"label":"","retainedMessages":1,"omittedMessages":0,"truncated":false}]}`},
		{"recall invalid truncated", `{"form":"recall","references":[{"label":"A","retainedMessages":1,"omittedMessages":0,"truncated":1}]}`},
		{"recall missing completeness", `{"form":"recall","references":[{"label":"A","retainedMessages":1,"omittedMessages":0}]}`},
		{"recall count string", `{"form":"recall","references":[{"label":"A","retainedMessages":"1","omittedMessages":0,"truncated":false}]}`},
		{"recall count null", `{"form":"recall","references":[{"label":"A","retainedMessages":null,"omittedMessages":0,"truncated":false}]}`},
		{"recall partial", `{"form":"recall","references":[{"label":"A","retainedMessages":1,"omittedMessages":0,"truncated":false},null]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := `{"kind":"producer","before":"retained before",` + strings.TrimPrefix(tc.source, "{")
			source = strings.TrimSuffix(source, "}") + `,"after":{"foreign":true}}`
			msg := contextBodyFixture(source, `[{"type":"text","text":"fallback model text"},{"type":"foreign","full":"block"}]`)
			original := append([]byte(nil), msg.Raw...)
			body := resolveContextBody(msg)
			if body.Form != "" || body.Summary != "" {
				t.Fatalf("malformed declaration specialized as %q", body.Form)
			}
			tt := contextBodyTester(body)
			for _, text := range []string{"fallback model text", "Unknown content block", "before", "retained before", "form", "after", `{"foreign":true}`} {
				contextBodyExpectText(t, tt, text)
			}
			if !bytes.Equal(msg.Raw, original) {
				t.Fatal("fallback changed retained raw fields")
			}
		})
	}
}

func TestContextBodyDeveloperEnvelopeAndUnavailableRaw(t *testing.T) {
	contextBodyEnglish(t)
	msg := contextBodyFixture(`{"form":"notice","summary":"nested summary"}`, `[{"type":"text","text":"nested content"}]`)
	msg.Raw = json.RawMessage(`{"type":"developer/message","data":{"source":{"form":"relay","senderSessionId":"outer ignored"},"content":[{"type":"text","text":"outer ignored"}],"message":{"source":{"form":"notice","summary":"nested summary"},"content":[{"type":"text","text":"nested content"}]}}}`)
	body := resolveContextBody(msg)
	if body.Form != "notice" || body.Summary != "nested summary" {
		t.Fatal("developer envelope did not read nested source")
	}
	tt := contextBodyTester(body)
	contextBodyExpectText(t, tt, "nested content")
	if tt.HasText("outer ignored") {
		t.Fatal("developer body read outer envelope fields")
	}
	for _, raw := range []json.RawMessage{nil, json.RawMessage(`{"invalid"`)} {
		msg.Raw, msg.Body = raw, "  retained projection\nwith whitespace  "
		tt = contextBodyTester(resolveContextBody(msg))
		contextBodyExpectText(t, tt, msg.Body)
	}
	msg = contextBodyFixture(`[]`, `[]`)
	tt = contextBodyTester(resolveContextBody(msg))
	if tt.HasText("LOSSY PROJECTION") || len(tt.Texts()) != 0 {
		t.Fatal("explicit empty content or nonrecord source invented fallback fields")
	}
}

func TestNoticeContextUsesOnlyContentRuns(t *testing.T) {
	contextBodyEnglish(t)
	msg := contextBodyFixture(`{"form":"snapshot","summary":"ignored summary","sections":[{"name":"ignored section","text":"ignored source text"}],"extra":"ignored metadata"}`, `[{"type":"text","text":"wake"},{"type":"text","text":" up"},{"type":"future","field":"retained block"},{"type":"text","text":"after"}]`)
	before := msg
	before.Raw = append(json.RawMessage(nil), msg.Raw...)
	tt := ui.NewTester(func(c *ui.Context) { paintNoticeContext(c, msg, contextBodyTestStyle()) }, 1000, 800)
	for _, text := range []string{"wake up", "Unknown content block", "after"} {
		contextBodyExpectText(t, tt, text)
	}
	if tt.HasText("ignored") || tt.HasText("Supersedes") {
		t.Fatal("trigger notice used source fields or specialized form")
	}
	if err := tt.Click("Unknown content block"); err != nil {
		t.Fatal(err)
	}
	contextBodyExpectText(t, tt, "{\n  \"type\": \"future\",\n  \"field\": \"retained block\"\n}")
	if !reflect.DeepEqual(msg, before) {
		t.Fatal("trigger painting mutated session.Message")
	}
}

func TestContextBodyForeignAndMalformedBlocksRemainWhole(t *testing.T) {
	contextBodyEnglish(t)
	for _, raw := range []string{
		`{"type":"text","text":null,"foreign":"retained"}`,
		`{"TYPE":"text","TEXT":"foreign"}`,
		`{"type":"future","Type":"text","text":"foreign"}`,
		`{"type":"future","extra":[1,{"nested":true}]}`,
		`["foreign array",{"x":1}]`, `null`, `27`, `"foreign primitive"`,
	} {
		t.Run(raw, func(t *testing.T) {
			msg := contextBodyFixture(`{"kind":"foreign"}`, `[{"type":"text","text":""},`+raw+`,{"type":"text","text":"after"}]`)
			tt := contextBodyTester(resolveContextBody(msg))
			contextBodyExpectText(t, tt, "Unknown content block")
			contextBodyExpectText(t, tt, "after")
			if err := tt.Click("Unknown content block"); err != nil {
				t.Fatal(err)
			}
			contextBodyExpectText(t, tt, contextFormattedJSON(json.RawMessage(raw)))
		})
	}
}

func TestContextBodyChineseDisclosureLabels(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(old) })
	msg := contextBodyFixture(`{"kind":"foreign"}`, `[{"type":"future","完整":"保留"}]`)
	tt := contextBodyTester(resolveContextBody(msg))
	if err := tt.Click("未知内容块"); err != nil {
		t.Fatal(err)
	}
	contextBodyExpectText(t, tt, "{\n  \"type\": \"future\",\n  \"完整\": \"保留\"\n}")
}

func TestContextBodyRecallNumericCopyMatchesReference(t *testing.T) {
	contextBodyEnglish(t)
	body := resolveContextBody(contextBodyFixture(`{"form":"recall","references":[{"label":"Numbers","retainedMessages":1e-7,"omittedMessages":-0,"truncated":false},{"label":"Overflow","retainedMessages":1e400,"omittedMessages":1e21,"truncated":false}]}`, `[]`))
	if body.Form != "recall" {
		t.Fatal("reference numeric values made recall fall back")
	}
	tt := contextBodyTester(body)
	contextBodyExpectText(t, tt, "1e-7 kept · 0 omitted")
	contextBodyExpectText(t, tt, "Infinity kept · 1e+21 omitted")
}

func TestContextBodyLeavesOuterPaddingAndHeightToOwner(t *testing.T) {
	contextBodyEnglish(t)
	text := strings.Repeat("line\n", 25) + "last"
	quoted, _ := json.Marshal(text)
	body := resolveContextBody(contextBodyFixture(`{"kind":"foreign"}`, `[{"type":"text","text":`+string(quoted)+`}]`))
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Padding(9, 13).Children(func() { body.paint(c, contextBodyTestStyle()) })
	}, 500, 900)
	r := contextBodyExpectText(t, tt, text)
	if r.X != 13 || r.Y != 9 || r.H <= 240 || r.W != 474 {
		t.Fatalf("body introduced an outer inset or height cap: %v", r)
	}
}

func TestContextBodyDisclosureStateStaysWithMessageAndBlock(t *testing.T) {
	contextBodyEnglish(t)
	msg := contextBodyFixture(`{"kind":"foreign"}`, `[{"type":"foreign","identity":"first"},{"type":"text","text":"between"},{"type":"foreign","identity":"second"}]`)
	body := resolveContextBody(msg)
	tt := ui.NewTester(func(c *ui.Context) { body.paint(c, contextBodyTestStyle()) }, 1000, 800)
	if err := tt.Click("Unknown content block"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText(`"identity": "first"`) || tt.HasText(`"identity": "second"`) {
		t.Fatal("opening one unknown block changed a sibling disclosure")
	}
	tt.Frame()
	if !tt.HasText(`"identity": "first"`) {
		t.Fatal("unknown block lost its disclosure state on a frame")
	}
	msg.ID = "another-message"
	body = resolveContextBody(msg)
	tt.Frame()
	if tt.HasText(`"identity": "first"`) || tt.HasText(`"identity": "second"`) {
		t.Fatal("JSON disclosure state crossed message scope")
	}
}
