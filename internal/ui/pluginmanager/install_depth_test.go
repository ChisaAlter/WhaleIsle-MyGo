package pluginmanager

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/svgimage"
)

func TestInstallANSIReplayPrecedesLineFolding(t *testing.T) {
	theme := ui.LightTheme()
	for _, tc := range []struct{ raw, want string }{
		{"100%\rOK", "OK0%"}, {"abc\b", "abc"}, {"abc\r\x1b[Kxy", "xy"}, {"界x\rA", "A x"}, {"e\u0301x\rA", "Ax"}, {"\x1b]0;inert title\x07a\n\x1b[0m", "a"},
	} {
		lines := installANSI(tc.raw, theme)
		var output []string
		for _, line := range lines {
			var text strings.Builder
			for _, span := range line {
				text.WriteString(span.Text)
			}
			output = append(output, text.String())
		}
		if got := strings.Join(output, "\n"); got != tc.want {
			t.Fatalf("%q => %q want %q", tc.raw, got, tc.want)
		}
	}
	lines := installANSI("\x1b[31mred\x1b[39mplain", theme)
	if len(lines) != 1 || len(lines[0]) != 2 || lines[0][0].Color != theme.Danger || lines[0][1].Color.A != 0 {
		t.Fatalf("SGR color or closer lost: %+v", lines)
	}
}

func TestInstallTerminalFoldsParsedLinesButCopiesRawOutput(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	var rows []string
	for n := 1; n <= 20; n++ {
		rows = append(rows, fmt.Sprintf("line-%02d", n))
	}
	raw := "\x1b[31m" + strings.Join(rows, "\n") + "\n\x1b[0m"
	tt := ui.NewTester(func(c *ui.Context) {
		installTerminal(c, manager.InstallRun{Command: "pnpm fixture", Output: raw, Settled: true})
	}, 600, 850)
	visible := strings.Join(tt.Texts(), "\n")
	if strings.Contains(visible, "line-10") || !strings.Contains(visible, "line-01") || !strings.Contains(visible, "line-20") {
		t.Fatal("head/tail cap did not use visible parsed lines", visible)
	}
	if err := tt.Click("Copy"); err != nil {
		t.Fatal(err)
	}
	if tt.Clipboard() != raw {
		t.Fatal("copy omitted hidden lines or rewrote source output")
	}
	if err := tt.Click("Expand 8 hidden lines"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(tt.Texts(), "\n"), "line-10") {
		t.Fatal("expand discarded the middle")
	}
}

type metadataIconFixture struct{ icon string }

func (f metadataIconFixture) Call(ctx context.Context, method string, args, out any) error {
	if method != "pluginManager/listBundles" {
		return (pageFixture{}).Call(ctx, method, args, out)
	}
	data, _ := json.Marshal([]manager.Bundle{{Name: "icon-fixture", Availability: "profile", Installed: true, Enabled: true, Meta: &manager.Meta{Title: manager.Text(`"Icon consumer"`), Icon: f.icon}}})
	return json.Unmarshal(data, out)
}
func TestActualManagerCardPassesSVGOwnerAndRetainsFallbackOnRetirement(t *testing.T) {
	source := `<svg xmlns="http://www.w3.org/2000/svg" width="30" height="30"><rect width="30" height="30" fill="red"><animate attributeName="opacity" values="0;1" dur="1s" repeatCount="indefinite"/></rect></svg>`
	calls := 0
	owner := &svgimage.Owner{ID: "retired-fixture", Context: context.Background(), Window: &mygo.Window{}, Valid: func() bool { calls++; return false }}
	controller := manager.New()
	controller.Attach(metadataIconFixture{icon: "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(source))}, nil)
	defer controller.Detach()
	if err := controller.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, Model{Controller: controller, SVGImages: owner}) }, 700, 500)
	if calls == 0 || !tt.HasText("Icon consumer") {
		t.Fatal("actual card did not consume the supplied image owner")
	}
	// Valid=false is observed before any helper startup. The source image seat
	// must return false so the existing default artwork can remain painted.
	if tt.Image().Bounds().Empty() {
		t.Fatal("card artwork failed to paint")
	}
}
