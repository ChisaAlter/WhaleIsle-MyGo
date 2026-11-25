package markdown

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestLinkActionsRenderVisibleCaptionAndInvokeCallback(t *testing.T) {
	var opened string
	links := []Link{
		{Label: "Project Docs", URL: "https://example.com/docs"},
	}
	st := Style{
		FontSize: 13,
		Accent:   ui.Hex("#2563eb"),
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Children(func() {
			LinkActions(c, links, st, func(u string) { opened = u })
		})
	}, 480, 200)

	want := "↗ Project Docs"
	if !tt.HasText(want) {
		t.Fatalf("link button rendered without visible caption %q: %v", want, tt.Texts())
	}
	if err := tt.Click(want); err != nil {
		t.Fatalf("click link button: %v", err)
	}
	if opened != "https://example.com/docs" {
		t.Fatalf("opened = %q, want %q", opened, "https://example.com/docs")
	}
}
