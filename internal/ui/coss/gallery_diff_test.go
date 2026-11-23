package coss

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type refCell struct {
	X, Y, W, H             float64
	PadL, PadR, PadT, PadB float64
	Radius, Border         float64
	BG, Color, Shadow      string
	Text                   *struct{ X, Y, W, H float64 }
}

type refSheet struct {
	Name  string             `json:"name"`
	Cells map[string]refCell `json:"cells"`
}

// cellResult is one specimen's comparison.
type cellResult struct {
	id                          string
	sheet                       string
	diffPct, meanDelta          float64
	dw, dh, dx, dy, dpad, dtext float64
	dRadius                     float64
	colorDelta                  float64
	refRadius, natRadius        float64
	hasText                     bool
}

func loadPNG(path string) (*image.RGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	if r, ok := img.(*image.RGBA); ok {
		return r, nil
	}
	b := img.Bounds()
	r := image.NewRGBA(b)
	draw.Draw(r, b, img, b.Min, draw.Src)
	return r, nil
}

func px(img *image.RGBA, x, y int) [3]int {
	if !(image.Point{x, y}.In(img.Bounds())) {
		return [3]int{}
	}
	o := img.PixOffset(x, y)
	return [3]int{int(img.Pix[o]), int(img.Pix[o+1]), int(img.Pix[o+2])}
}

func delta(a, b [3]int) int {
	d := 0
	for i := range a {
		d = max(d, abs(a[i]-b[i]))
	}
	return d
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// cornerRadius estimates a box's top-left radius from pixels: the first
// pixel along the corner's diagonal that differs from the page by at least
// half the box's own contrast lies at r(1 − 1/√2).
func cornerRadius(img *image.RGBA, x, y, w, h float64) float64 {
	x0, y0 := int(math.Round(x)), int(math.Round(y))
	page := px(img, x0-3, y0+int(h/2))
	mid := px(img, x0+1, y0+int(h/2))
	full := delta(page, mid)
	if full < 12 {
		return math.NaN() // no visible box edge (ghost, link)
	}
	limit := int(min(w, h) / 2)
	for k := 0; k <= limit; k++ {
		if delta(px(img, x0+k, y0+k), page)*2 >= full {
			return float64(k) / (1 - 1/math.Sqrt2)
		}
	}
	return math.NaN()
}

func TestGalleryDiff(t *testing.T) {
	dir := os.Getenv("COSS_GALLERY_DIR")
	if dir == "" {
		t.Skip("set COSS_GALLERY_DIR (after TestGalleryRender and ref.mjs) to diff the gallery")
	}
	var m manifestJSON
	var ref struct {
		Sheets []refSheet `json:"sheets"`
	}
	for path, v := range map[string]any{"manifest.json": &m, "ref-geometry.json": &ref} {
		data, err := os.ReadFile(filepath.Join(dir, path))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, v); err != nil {
			t.Fatal(err)
		}
	}
	refBy := map[string]refSheet{}
	for _, s := range ref.Sheets {
		refBy[s.Name] = s
	}
	sideDir := filepath.Join(dir, "side")
	os.MkdirAll(sideDir, 0o755)
	byComp := map[string][]cellResult{}
	var comps []string
	missing := map[string]int{}
	for _, sh := range m.Sheets {
		rs, ok := refBy[sh.Name]
		if !ok {
			missing[sh.Comp] += len(sh.Cells)
			continue
		}
		nat, err := loadPNG(filepath.Join(dir, "native", sh.Name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		rimg, err := loadPNG(filepath.Join(dir, "ref", sh.Name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		if _, seen := byComp[sh.Comp]; !seen {
			comps = append(comps, sh.Comp)
		}
		heat := image.NewRGBA(nat.Bounds())
		for _, cell := range sh.Cells {
			rc, ok := rs.Cells[cell.ID]
			if !ok || cell.HTML == "" {
				missing[sh.Comp]++
				continue
			}
			r := cellResult{id: cell.ID, sheet: sh.Name}
			n, sum, diff := 0, 0, 0
			for y := int(cell.Y); y < int(cell.Y+cell.H); y++ {
				for x := int(cell.X); x < int(cell.X+cell.W); x++ {
					d := delta(px(nat, x, y), px(rimg, x, y))
					n++
					sum += d
					if d > 32 {
						diff++
						heat.Set(x, y, color.RGBA{255, 0, 0, 255})
					} else if d > 8 {
						heat.Set(x, y, color.RGBA{255, 200, 0, 255})
					} else {
						g := px(nat, x, y)
						heat.Set(x, y, color.RGBA{uint8(g[0] / 3), uint8(g[1] / 3), uint8(g[2] / 3), 255})
					}
				}
			}
			if n > 0 {
				r.diffPct = 100 * float64(diff) / float64(n)
				r.meanDelta = float64(sum) / float64(n)
			}
			nb := cell.Native
			r.dw = float64(nb.W) - rc.W
			r.dh = float64(nb.H) - rc.H
			r.dx = float64(nb.X) - rc.X
			r.dy = float64(nb.Y) - rc.Y
			if cell.Text != nil && rc.Text != nil {
				r.hasText = true
				r.dpad = (float64(cell.Text.X) - float64(nb.X)) - (rc.Text.X - rc.X)
				// Line boxes differ (fixed coss line height vs the font's
				// own); compare the text's vertical centers.
				r.dtext = (float64(cell.Text.Y) + float64(cell.Text.H)/2) - (rc.Text.Y + rc.Text.H/2)
			}
			r.natRadius = cornerRadius(nat, float64(nb.X), float64(nb.Y), float64(nb.W), float64(nb.H))
			r.refRadius = cornerRadius(rimg, rc.X, rc.Y, rc.W, rc.H)
			if !math.IsNaN(r.natRadius) && !math.IsNaN(r.refRadius) {
				r.dRadius = r.natRadius - r.refRadius
			}
			// Fill color just inside the left padding, mid-height.
			sx, sy := int(math.Round(rc.X))+3, int(math.Round(rc.Y+rc.H/2))
			r.colorDelta = float64(delta(px(nat, sx, sy), px(rimg, sx, sy)))
			byComp[sh.Comp] = append(byComp[sh.Comp], r)
			if z := os.Getenv("COSS_GALLERY_ZOOM"); z != "" && strings.Contains(cell.ID+"("+sh.Scheme+")", z) {
				zoom(t, filepath.Join(dir, "zoom"), cell, sh.Scheme, nat, rimg)
			}
		}
		// Side by side: native | reference | heat map.
		b := nat.Bounds()
		side := image.NewRGBA(image.Rect(0, 0, b.Dx()*3+16, b.Dy()))
		draw.Draw(side, side.Bounds(), image.NewUniform(color.RGBA{128, 128, 128, 255}), image.Point{}, draw.Src)
		draw.Draw(side, b, nat, image.Point{}, draw.Src)
		draw.Draw(side, b.Add(image.Pt(b.Dx()+8, 0)), rimg, image.Point{}, draw.Src)
		draw.Draw(side, b.Add(image.Pt(2*b.Dx()+16, 0)), heat, image.Point{}, draw.Src)
		writePNG(t, filepath.Join(sideDir, sh.Name+".png"), side)
	}
	sort.Strings(comps)
	var rep strings.Builder
	fmt.Fprintf(&rep, "# coss gallery diff\n\nNative (MyGo headless, scale 1) vs coss markup in headless Chromium, same font stack, same accent. ")
	fmt.Fprintf(&rep, "diff%% = share of cell pixels whose max channel delta > 32; geometry deltas are native − reference in DIPs ")
	fmt.Fprintf(&rep, "(w/h = box size, pad = text inset from the box's left edge, ty = text vertical center, r = top-left radius estimated from pixels, c = fill delta just inside the left edge, 0–255).\n\n")
	fmt.Fprintf(&rep, "| component | specimens | median diff%% | max diff%% | max abs w | max abs h | max abs pad | max abs ty | max abs r | max c | geometry > 1 DIP |\n|---|---|---|---|---|---|---|---|---|---|---|\n")
	var details strings.Builder
	for _, comp := range comps {
		rs := byComp[comp]
		if len(rs) == 0 {
			continue
		}
		diffs := make([]float64, len(rs))
		var mw, mh, mp, mt, mr, mc, md float64
		var off []string
		for i, r := range rs {
			diffs[i] = r.diffPct
			md = max(md, r.diffPct)
			mw, mh = max(mw, math.Abs(r.dw)), max(mh, math.Abs(r.dh))
			mp, mt = max(mp, math.Abs(r.dpad)), max(mt, math.Abs(r.dtext))
			mr, mc = max(mr, math.Abs(r.dRadius)), max(mc, r.colorDelta)
			if math.Abs(r.dw) > 1 || math.Abs(r.dh) > 1 || math.Abs(r.dpad) > 1 || math.Abs(r.dRadius) > 1.5 {
				off = append(off, r.id)
			}
		}
		sort.Float64s(diffs)
		fmt.Fprintf(&rep, "| %s | %d | %.2f | %.2f | %.1f | %.1f | %.1f | %.1f | %.1f | %.0f | %d |\n", comp, len(rs), diffs[len(diffs)/2], md, mw, mh, mp, mt, mr, mc, len(off))
		fmt.Fprintf(&details, "\n## %s\n\n| specimen (scheme) | diff%% | w | h | pad | ty | r (nat/ref) | c |\n|---|---|---|---|---|---|---|---|\n", comp)
		for _, r := range rs {
			scheme := strings.TrimPrefix(r.sheet, comp+"-")
			fmt.Fprintf(&details, "| %s (%s) | %.2f | %+.1f | %+.1f | %+.1f | %+.1f | %.1f/%.1f | %.0f |\n", r.id, scheme, r.diffPct, r.dw, r.dh, r.dpad, r.dtext, r.natRadius, r.refRadius, r.colorDelta)
		}
	}
	if len(missing) > 0 {
		fmt.Fprintf(&rep, "\nSpecimens without a reference (native only):")
		var ks []string
		for k := range missing {
			ks = append(ks, k)
		}
		sort.Strings(ks)
		for _, k := range ks {
			fmt.Fprintf(&rep, " %s (%d)", k, missing[k])
		}
		rep.WriteString("\n")
	}
	rep.WriteString(details.String())
	if err := os.WriteFile(filepath.Join(dir, "report.md"), []byte(rep.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("report: %s", filepath.Join(dir, "report.md"))
}

// zoom writes one specimen at 4× as native | reference.
func zoom(t *testing.T, dir string, cell cellJSON, scheme string, nat, rimg *image.RGBA) {
	os.MkdirAll(dir, 0o755)
	const k = 4
	w, h := int(cell.W), int(cell.H)
	out := image.NewRGBA(image.Rect(0, 0, (2*w+2)*k, h*k))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			a := px(nat, int(cell.X)+x, int(cell.Y)+y)
			b := px(rimg, int(cell.X)+x, int(cell.Y)+y)
			for dy := 0; dy < k; dy++ {
				for dx := 0; dx < k; dx++ {
					out.Set(x*k+dx, y*k+dy, color.RGBA{uint8(a[0]), uint8(a[1]), uint8(a[2]), 255})
					out.Set((w+2+x)*k+dx, y*k+dy, color.RGBA{uint8(b[0]), uint8(b[1]), uint8(b[2]), 255})
				}
			}
		}
	}
	writePNG(t, filepath.Join(dir, strings.NewReplacer("/", "_", "+", "-").Replace(cell.ID)+"-"+scheme+".png"), out)
}
