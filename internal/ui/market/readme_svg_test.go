package market

import (
	"context"
	"fmt"
	"image/color"
	"os"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/svgimage"
)

func TestMarketReadmeSVGCapabilityRetainsSupportedStaticDocuments(t *testing.T) {
	for _, content := range []string{
		`<rect width="40" height="20" fill="#4176e6"/>`,
		`<defs><linearGradient id="g"><stop offset="0" stop-color="red"/><stop offset="1" stop-color="blue"/></linearGradient></defs><rect width="40" height="20" fill="url('#g')"/>`,
		`<style>rect.node, #square { fill: rgb(65, 118, 230); stroke: none; }</style><rect id="square" class="node" width="40" height="20"/>`,
		`<metadata><editor:document xmlns:editor="urn:editor"><editor:text>not painted</editor:text></editor:document></metadata><title>Shape</title><desc>Accessible description</desc><rect width="40" height="20"/>`,
		`<defs><mask id="m"><rect width="40" height="20" fill="white"/></mask><symbol id="s" viewBox="0 0 40 20"><rect width="40" height="20"/></symbol></defs><use href="#s" width="40" height="20" mask="url(#m)"/>`,
	} {
		source := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 40 20">` + content + `</svg>`
		if !supportedReadmeSVG([]byte(source)) {
			t.Errorf("supported static SVG rejected: %s", content)
		}
	}
}

func TestMarketReadmeSVGUnconditionalSwitchRendersItsFirstChild(t *testing.T) {
	cache := readmeImageCache(t)
	// Explicit intrinsic dimensions keep this fixture on the static path.
	// A viewBox alone now correctly needs browser natural-size metadata.
	const source = `<svg width='40' height='20' viewBox='0 0 40 20'><switch><rect width='40' height='20' fill='red'/><rect width='40' height='20' fill='blue'/></switch></svg>`
	const url = "https://example.invalid/unconditional-switch.svg"
	if err := os.WriteFile(cache.FilePath(url), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			marketReadme(c, Model{}, "![Switch diagram]("+url+")", c.Theme().Text, c.Theme().TextMuted, c.Theme().Surface, c.Theme().Accent)
		})
	}, 420, 300)
	r, ok := tt.Find("README image " + url)
	if !ok || r.W != 40 || r.H != 20 || tt.HasText("Switch diagram") {
		t.Fatalf("unconditional switch did not render at its natural size: %+v", r)
	}
	if pixel := tt.Image().RGBAAt(int(r.X+r.W/2), int(r.Y+r.H/2)); pixel != (color.RGBA{R: 255, A: 255}) {
		t.Fatalf("switch's first rectangle was not painted red: %v", pixel)
	}
	for _, attr := range []string{"requiredFeatures", "requiredExtensions", "systemLanguage"} {
		conditional := strings.Replace(source, "<rect ", "<rect "+attr+"='unknown' ", 1)
		if supportedReadmeSVG([]byte(conditional)) {
			t.Errorf("switch condition %s bypassed the attribute capability gate", attr)
		}
	}
}

func TestMarketReadmeBrowserSVGRevokedOwnerRetainsAltAndParagraph(t *testing.T) {
	cache := readmeImageCache(t)
	const source = `<svg width="800" height="400"><animate attributeName="opacity" values="0;1" dur="1s"/></svg>`
	const url = "https://example.invalid/revoked-owner.svg"
	if err := os.WriteFile(cache.FilePath(url), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	owner := &svgimage.Owner{ID: "revoked", Window: &mygo.Window{}, Context: context.Background(), Valid: func() bool { return false }}
	tt := ui.NewTester(func(c *ui.Context) {
		marketReadme(c, Model{SVGImages: owner}, "![Authored diagram]("+url+")\n\nFollowing paragraph", c.Theme().Text, c.Theme().TextMuted, c.Theme().Surface, c.Theme().Accent)
	}, 420, 600)
	if !tt.HasText("Authored diagram") || !tt.HasText("Following paragraph") {
		t.Fatal("failed image authority swallowed authored fallback or following prose")
	}
	if r, _ := tt.Find("README image " + url); r.W != 0 || r.H != 0 {
		t.Fatalf("revoked image retained a phantom canvas: %+v", r)
	}
}

func TestMarketReadmeUnsupportedSVGKeepsAuthoredAltInsteadOfPartialImage(t *testing.T) {
	cache := readmeImageCache(t)
	for name, content := range map[string]string{
		"embedded-image":  `<image href="data:image/png;base64,AAAA" width="40" height="20"/>`,
		"animation":       `<animate attributeName="opacity" from="0" to="1" dur="1s" repeatCount="indefinite"/>`,
		"foreign-object":  `<foreignObject width="40" height="20"><div xmlns="http://www.w3.org/1999/xhtml">Label</div></foreignObject>`,
		"css-filter":      `<style>rect {filter:blur(4px)}</style>`,
		"inline-filter":   `<g style="filter:blur(4px)"><circle r="4"/></g>`,
		"marker-property": `<path d="M0 0L40 20" marker-end="url(#arrow)"/>`,
		"media-css":       `<style>@media (prefers-color-scheme:dark) {rect {fill:red}}</style>`,
		"css-variable":    `<rect width="40" height="20" fill="var(--diagram-color,red)"/>`,
		"external-use":    `<use href="https://example.invalid/shapes.svg#shape"/>`,
		"external-paint":  `<rect width="40" height="20" fill="url(https://example.invalid/palette.svg#color)"/>`,
	} {
		t.Run(name, func(t *testing.T) {
			// A visible rectangle would otherwise make ParseSVG's incomplete
			// rendering look successful, even when essential content is absent.
			source := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 400"><rect width="800" height="400" fill="#4176e6"/>` + content + `</svg>`
			if _, err := ui.ParseSVG([]byte(source)); err != nil {
				t.Fatalf("fixture must exercise the permissive native parser: %v", err)
			}
			url := "https://example.invalid/" + name + ".svg"
			if err := os.WriteFile(cache.FilePath(url), []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			alt := "Authored diagram description " + name
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() {
					marketReadme(c, Model{}, fmt.Sprintf("![%s](%s)\n\nFollowing paragraph", alt, url), c.Theme().Text, c.Theme().TextMuted, c.Theme().Surface, c.Theme().Accent)
				})
			}, 420, 600)
			if !tt.HasText(alt) || !tt.HasText("Following paragraph") {
				t.Fatal("unsupported SVG lost authored alt or following text")
			}
			picture, _ := tt.Find("README image " + url)
			if picture.H != 0 || picture.W != 0 {
				t.Fatalf("unsupported SVG mounted a partial native image: %+v", picture)
			}
			for _, text := range tt.Texts() {
				if strings.TrimSpace(text) == alt {
					r, ok := tt.Find(text)
					if !ok || r.H > 48 {
						t.Fatalf("alt retained a phantom canvas: %+v", r)
					}
				}
			}
		})
	}
}
