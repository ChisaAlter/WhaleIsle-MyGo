package chat

import (
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

func toolReadViewCall(n int) *session.ToolCallView {
	lines := make([]map[string]any, n)
	for i := range lines {
		lines[i] = map[string]any{"number": 37 + i*2, "text": fmt.Sprintf("recorded-line-%02d", i)}
	}
	return fileToolFixture("read", `{"file_path":"/work/src/read.go","offset":37}`, map[string]any{"path": "/work/src/read.go", "offset": 37, "lines": lines, "totalLines": 180, "lang": ""}, "<path>/work/src/read.go</path>\n<type>file</type>\n<content>\nrecorded envelope, not displayed\n</content>")
}
func openReadView(t *testing.T, tt *ui.Tester, expanded *map[string]bool) {
	t.Helper()
	(*expanded)["summary|tool:file-call"] = true
	tt.Frame()
	tt.SetTime(time.Unix(11700, 0).Add(250 * time.Millisecond))
	tt.Frame()
}
func TestToolReadViewUsesSingleTypedCardAndWholeWindowCopy(t *testing.T) {
	for _, dark := range []bool{false, true} {
		t.Run(fmt.Sprint(dark), func(t *testing.T) {
			call := toolReadViewCall(14)
			tt, expanded, opened := toolSummaryViewFixture(t, call, 640, dark, false)
			openReadView(t, tt, expanded)
			for _, absent := range []string{"IN", "OUT", "Path", `"file_path"`, "recorded envelope", ": recorded-line"} {
				if tt.HasText(absent) {
					t.Fatalf("duplicate %s %v", absent, tt.Texts())
				}
			}
			for _, want := range []string{"Read content card", "Showing 14 of 180 lines", "Expand 6 more lines", "recorded-line-00", "recorded-line-13"} {
				if _, ok := tt.Find(want); !ok {
					t.Fatalf("missing%s %v", want, tt.Texts())
				}
			}
			if tt.HasText("recorded-line-06") {
				t.Fatal("middle not folded")
			}
			card, _ := tt.Find("Read content card")
			toolTreeNear(t, "read inset", card.X, 24)
			toolTreeNear(t, "read folded height", card.H, 239)
			copyButton, _ := tt.Find("Copy")
			toolTreeNear(t, "read action width", copyButton.W, 24)
			toolTreeNear(t, "read action height", copyButton.H, 24)
			language, _ := tt.Find("Code block")
			toolTreeNear(t, "read language inset", language.X-card.X, 22)
			toolReadExport(t, tt, fmt.Sprintf("folded-dark-%v", dark))
			if err := tt.Click("Copy"); err != nil {
				t.Fatal(err)
			}
			var parts []string
			for i := 0; i < 14; i++ {
				parts = append(parts, fmt.Sprintf("recorded-line-%02d", i))
			}
			if tt.Clipboard() != strings.Join(parts, "\n") {
				t.Fatalf("read copied envelope/gutter%q", tt.Clipboard())
			}
			tt.SetClipboard("copy-guard")
			tt.Key(0, ui.KeySpace)
			if tt.Clipboard() != "copy-guard" {
				t.Fatal("copied state repeated write")
			}
			tt.SetTime(time.Unix(11702, 0))
			tt.Key(0, ui.KeyEnter)
			if tt.Clipboard() != strings.Join(parts, "\n") {
				t.Fatal("copied timeout failed")
			}
			if err := tt.Click("Expand 6 more lines"); err != nil {
				t.Fatal(err)
			}
			if !tt.HasText("recorded-line-06") {
				t.Fatal("expand failed")
			}
			card, _ = tt.Find("Read content card")
			toolTreeNear(t, "read expanded height", card.H, 353)
			toolReadExport(t, tt, fmt.Sprintf("expanded-dark-%v", dark))
			tt.Key(0, ui.KeySpace)
			if tt.HasText("recorded-line-06") {
				t.Fatal("focused fold failed")
			}
			if len(*opened) > 0 {
				t.Fatal("body opened file")
			}
		})
	}
}
func TestToolReadViewMalformedPendingAndReadOnlyKeepTheirOwningBodies(t *testing.T) {
	invalid := toolReadViewCall(3)
	invalid.Result.Raw = json.RawMessage(strings.ReplaceAll(string(invalid.Result.Raw), `"totalLines":180`, `"totalLines":"invalid"`))
	tt, expanded, _ := toolSummaryViewFixture(t, invalid, 640, false, false)
	openReadView(t, tt, expanded)
	if tt.HasText("Read content card") || tt.HasText("IN") || !tt.HasText("OUT") || !tt.HasText("recorded envelope") {
		t.Fatalf("invalidmeta%v", tt.Texts())
	}
	pending := toolReadViewCall(3)
	pending.Result = nil
	pending.State = "running"
	tt, _, _ = toolSummaryViewFixture(t, pending, 640, false, false)
	tt.ClickAt(610, 32)
	tt.Frame()
	if tt.HasText("IN") || tt.HasText("Path") || tt.HasText("Read content card") {
		t.Fatalf("pending acquired body%v", tt.Texts())
	}
	ro := toolReadViewCall(3)
	tt, expanded, _ = toolSummaryViewFixture(t, ro, 640, false, true)
	openReadView(t, tt, expanded)
	if tt.HasText("Read content card") {
		t.Fatal("readonly acquired active preset")
	}
	if !tt.HasText("recorded envelope") {
		t.Fatal("readonly lost receipt")
	}
}
func toolReadExport(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("TOOL_READ_EVIDENCE_DIR")
	if dir == "" {
		return
	}
	prefix := filepath.Join(dir, "tool-read-pass262-native-software-"+name)
	file, err := os.Create(prefix + ".png")
	if err != nil {
		t.Fatal(err)
	}
	if err = png.Encode(file, tt.Image()); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	boxes := map[string]ui.Rect{}
	for _, label := range []string{"Read content card", "Code block", "Showing 14 of 180 lines", "Copy", "Wrap lines", "Expand 6 more lines", "Collapse content", "recorded-line-00", "recorded-line-13", "Following summary flow"} {
		if r, ok := tt.Find(label); ok {
			boxes[label] = r
		}
	}
	value, _ := json.MarshalIndent(map[string]any{"name": name, "rects": boxes, "texts": tt.Texts()}, "", "  ")
	if err = os.WriteFile(prefix+".json", value, 0600); err != nil {
		t.Fatal(err)
	}
}
