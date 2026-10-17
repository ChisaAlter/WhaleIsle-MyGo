package chat

import (
	"encoding/json"
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/store/session"
)

func TestTrajectoryStructuredInspector(t *testing.T) {
	raw := `{"nested":{"a.b":[9007199254740993,"full text"]},"flag":true}`
	selected, tab, query := "tool", "args", ""
	tt := ui.NewTester(func(c *ui.Context) {
		TrajectoryView(c, TrajectoryModel{Snapshot: session.ViewState{SelectedID: "s", Messages: []session.Message{{ID: "tool", Kind: session.MessageKindTool, Args: raw}}}, SelectedID: &selected, DetailTab: &tab, SearchQuery: &query})
	}, 1000, 1200)
	if err := tt.Click("Structured JSON"); err != nil {
		t.Fatal("structured field inspection absent:", err)
	}
	if err := tt.Click("Expand JSON $.nested"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(`Expand JSON $.nested["a.b"]`); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(`Copy JSON path $.nested["a.b"][0]`); err != nil {
		t.Fatal(err)
	}
	if tt.Clipboard() != `$.nested["a.b"][0]` {
		t.Fatal(tt.Clipboard())
	}
	if err := tt.Click(`Copy JSON value $.nested["a.b"][0]`); err != nil {
		t.Fatal(err)
	}
	if tt.Clipboard() != "9007199254740993" {
		t.Fatal("large integer changed")
	}
}

func TestTrajectoryJSONFallbackAndCompleteCopy(t *testing.T) {
	for _, raw := range []string{"plain text", "null", "123", `{"a":1} {"b":2}`} {
		if _, ok := trajectoryJSONValue(raw); ok {
			t.Fatalf("not a single JSON container: %s", raw)
		}
	}
	text := strings.Repeat("long ", 1000) + "TAIL"
	rawBytes, _ := json.Marshal(map[string]string{"field": text})
	body := string(rawBytes)
	scope := "one"
	tt := ui.NewTester(func(c *ui.Context) { trajectoryPayloadView(c, scope, body, ui.RGB(0, 0, 0), ui.RGB(80, 80, 80)) }, 800, 800)
	tt.Click("Structured JSON")
	tt.Click("Copy JSON value $.field")
	if tt.Clipboard() != text {
		t.Fatal("field value truncated")
	}
	tt.Click("Copy JSON pretty $")
	var parsed map[string]string
	if json.Unmarshal([]byte(tt.Clipboard()), &parsed) != nil || parsed["field"] != text {
		t.Fatal("pretty subtree copy lost data")
	}
	scope = "two"
	tt.Frame()
	if _, ok := tt.Find("Structured JSON"); !ok {
		t.Fatal("tree state crossed session/record")
	}
	if !tt.HasText(body) {
		t.Fatal("raw fallback not complete")
	}
}
