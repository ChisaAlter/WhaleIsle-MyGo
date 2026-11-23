package coss

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"
)

// The component gallery renders every component × variant × size × state in
// light and dark to PNG sheets, for comparison with coss's own markup
// rendered by headless Chromium (scripts/coss-gallery). It runs only when
// COSS_GALLERY_DIR names a directory:
//
//	COSS_GALLERY_DIR=tmp/coss-gallery/run go test ./internal/ui/coss -run TestGalleryRender
//	node scripts/coss-gallery/ref.mjs tmp/coss-gallery/run globals.css
//	COSS_GALLERY_DIR=tmp/coss-gallery/run go test ./internal/ui/coss -run TestGalleryDiff

// specimen is one cell of a sheet: a component in one variant, size and
// state, with coss's markup for the same thing. The html may use
// {{cva:component.cvaName:k=v,…}} and {{part:component/slot[@config]}}
// placeholders, which ref.mjs resolves from docs/design/coss/index.json; the
// element marked data-g (else the cell's first element) is measured.
type specimen struct {
	id    string
	w, h  float32
	state State
	label string
	html  string
	build func(c *ui.Context) ui.Element
}

type sheet struct {
	comp  string
	specs []specimen
}

var gallerySheets []func() sheet

func registerSheet(f func() sheet) bool {
	gallerySheets = append(gallerySheets, f)
	return true
}

const (
	galleryPad   = 8
	galleryWidth = 1280
)

// Gallery accents: the Host theme's --dsw-alias-state-business-primary in
// each half (internal/ui/theme deepseekLight/deepseekDark Accent).
var galleryAccent = map[bool]ui.Color{false: ui.Hex("#4176e6"), true: ui.Hex("#7aaaff")}

type rectJSON struct {
	X, Y, W, H float32
}

type cellJSON struct {
	ID     string    `json:"id"`
	X      float32   `json:"x"`
	Y      float32   `json:"y"`
	W      float32   `json:"w"`
	H      float32   `json:"h"`
	HTML   string    `json:"html,omitempty"`
	Label  string    `json:"label,omitempty"`
	Native rectJSON  `json:"native"`
	Text   *rectJSON `json:"nativeText,omitempty"`
}

type sheetJSON struct {
	Name   string            `json:"name"`
	Comp   string            `json:"comp"`
	Scheme string            `json:"scheme"`
	Width  int               `json:"width"`
	Height int               `json:"height"`
	Font   string            `json:"font"`
	Pad    int               `json:"pad"`
	Vars   map[string]string `json:"-"`
	Style  string            `json:"rootStyle"`
	Cells  []cellJSON        `json:"cells"`
}

type manifestJSON struct {
	Sheets []sheetJSON `json:"sheets"`
}

func hexOf(c ui.Color) string {
	if c.A == 255 {
		return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
	}
	return fmt.Sprintf("#%02x%02x%02x%02x", c.R, c.G, c.B, c.A)
}

// layout places specs in rows that wrap at galleryWidth.
func layout(specs []specimen) (pos [][2]float32, w, h int) {
	x, y, rowH := float32(0), float32(0), float32(0)
	for _, s := range specs {
		if x > 0 && x+s.w > galleryWidth {
			x, y, rowH = 0, y+rowH, 0
		}
		pos = append(pos, [2]float32{x, y})
		x += s.w
		rowH = max(rowH, s.h)
	}
	return pos, galleryWidth, int(y + rowH)
}

func TestGalleryRender(t *testing.T) {
	dir := os.Getenv("COSS_GALLERY_DIR")
	if dir == "" {
		t.Skip("set COSS_GALLERY_DIR to render the coss gallery")
	}
	only := os.Getenv("COSS_GALLERY_ONLY")
	if err := os.MkdirAll(filepath.Join(dir, "native"), 0o755); err != nil {
		t.Fatal(err)
	}
	font := theme.TextFamily("")
	var m manifestJSON
	for _, mk := range gallerySheets {
		sh := mk()
		if only != "" && !strings.Contains(","+only+",", ","+sh.comp+",") {
			continue
		}
		pos, w, h := layout(sh.specs)
		for _, dark := range []bool{false, true} {
			scheme := "light"
			if dark {
				scheme = "dark"
			}
			bounds := make([]ui.Rect, len(sh.specs))
			tt := ui.NewTester(func(c *ui.Context) {
				th := *c.Theme()
				th.Dark, th.Font, th.FontSize, th.Accent = dark, font, 14, galleryAccent[dark]
				c.SetTheme(&th)
				tk := design.For(c)
				ui.Box(c).Fill().Background(tk.Background).TextColor(tk.Foreground).Children(func() {
					for i, s := range sh.specs {
						ui.Row(c).Key(s.id).Absolute().Left(pos[i][0]).Top(pos[i][1]).Size(s.w, s.h).Padding(galleryPad).
							AlignItems(ui.Start).Justify(ui.Start).Children(func() {
							forced = s.state
							root := s.build(c)
							forced = 0
							bounds[i] = root.Bounds()
						})
					}
				})
			}, w, max(h, 1))
			tt.SetDark(dark)
			tt.Frame()
			tt.Frame()
			name := sh.comp + "-" + scheme
			writePNG(t, filepath.Join(dir, "native", name+".png"), tt.Image())
			tk := design.Lookup(map[bool]design.Scheme{false: design.Light, true: design.Dark}[dark], galleryAccent[dark], ui.Color{})
			sj := sheetJSON{Name: name, Comp: sh.comp, Scheme: scheme, Width: w, Height: h, Font: font, Pad: galleryPad,
				Style: fmt.Sprintf("--primary:%s;--primary-foreground:%s;--ring:%s", hexOf(tk.Primary), hexOf(tk.PrimaryForeground), hexOf(tk.Ring))}
			for i, s := range sh.specs {
				b := bounds[i]
				cj := cellJSON{ID: s.id, X: pos[i][0], Y: pos[i][1], W: s.w, H: s.h, HTML: s.html, Label: s.label,
					Native: rectJSON{b.X, b.Y, b.W, b.H}}
				if s.label != "" {
					for _, r := range tt.FindAll(s.label) {
						if r.X >= b.X-1 && r.Y >= b.Y-1 && r.X+r.W <= b.X+b.W+1 && r.Y+r.H <= b.Y+b.H+1 {
							cj.Text = &rectJSON{r.X, r.Y, r.W, r.H}
							break
						}
					}
				}
				sj.Cells = append(sj.Cells, cj)
			}
			m.Sheets = append(m.Sheets, sj)
		}
	}
	data, _ := json.MarshalIndent(m, "", " ")
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, s := range m.Sheets {
		n += len(s.Cells)
	}
	t.Logf("rendered %d sheets, %d specimens", len(m.Sheets), n)
}

func writePNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

// TestGalleryCoverage keeps every sheet's specimens unique and buildable
// headlessly without COSS_GALLERY_DIR: the gallery is also a smoke test of
// every component in every state.
func TestGalleryCoverage(t *testing.T) {
	seen := map[string]bool{}
	var comps []string
	total := 0
	for _, mk := range gallerySheets {
		sh := mk()
		comps = append(comps, sh.comp)
		for _, s := range sh.specs {
			if seen[s.id] {
				t.Fatalf("duplicate specimen %s", s.id)
			}
			seen[s.id] = true
			total++
		}
		pos, w, h := layout(sh.specs)
		for _, dark := range []bool{false, true} {
			ui.NewTester(func(c *ui.Context) {
				th := *c.Theme()
				th.Dark, th.Accent = dark, galleryAccent[dark]
				c.SetTheme(&th)
				ui.Box(c).Fill().Children(func() {
					for i, s := range sh.specs {
						ui.Row(c).Key(s.id).Absolute().Left(pos[i][0]).Top(pos[i][1]).Size(s.w, s.h).AlignItems(ui.Start).Children(func() {
							forced = s.state
							s.build(c)
							forced = 0
						})
					}
				})
			}, w, max(h, 1))
		}
	}
	sort.Strings(comps)
	t.Logf("%d components, %d specimens: %s", len(comps), total, strings.Join(comps, " "))
}

// stateName names the forced states of a specimen.
func stateName(s State) string {
	if s == 0 {
		return "default"
	}
	var parts []string
	for _, x := range []struct {
		s State
		n string
	}{{StateHover, "hover"}, {StatePressed, "pressed"}, {StateFocusVisible, "focus"}, {StateDisabled, "disabled"},
		{StateInvalid, "invalid"}, {StateOpen, "open"}, {StateChecked, "checked"}, {StateIndeterminate, "indeterminate"},
		{StateSelected, "selected"}, {StateHighlighted, "highlighted"}, {StateLoading, "loading"}} {
		if s.Has(x.s) {
			parts = append(parts, x.n)
		}
	}
	return strings.Join(parts, "+")
}

// force returns the data-force attribute for a state's pseudo-classes, and
// the data attributes coss styles the rest by.
func force(s State) string {
	var ps []string
	if s.Has(StateHover) {
		ps = append(ps, "hover")
	}
	if s.Has(StatePressed) {
		ps = append(ps, "active")
	}
	if s.Has(StateFocusVisible) {
		ps = append(ps, "focus", "focus-visible", "focus-within")
	}
	out := ""
	if len(ps) > 0 {
		out = ` data-force="` + strings.Join(ps, ",") + `"`
	}
	if s.Has(StatePressed) {
		out += ` data-pressed=""`
	}
	return out
}

// iconHTML is the lucide-react markup of i.
func iconHTML(i Icon) string {
	if i == IconNone {
		return ""
	}
	return `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="` + i.stroke() + `" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">` + iconPaths[i] + `</svg>`
}

var baseStates = []State{0, StateHover, StatePressed, StateFocusVisible, StateDisabled}
