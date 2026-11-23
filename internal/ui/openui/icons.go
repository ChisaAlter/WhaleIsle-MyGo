package openui

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"io"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
)

// lucide.json.gz is the Lucide icon set (ISC) the Whale UI viewer bundles,
// extracted by gen/lucide.mjs: PascalCase name → SVG inner markup.
//
//go:embed lucide.json.gz
var lucideGz []byte

var (
	lucideOnce  sync.Once
	lucideMu    sync.Mutex
	lucideSrc   map[string]string
	lucideCache = map[string]*ui.SVG{}
)

// lucideAliases are the older names the viewer resolves besides the
// canonical set.
var lucideAliases = map[string]string{
	"CheckCircle":  "CircleCheckBig",
	"CheckCircle2": "CircleCheck",
	"AlertCircle":  "CircleAlert",
	"HelpCircle":   "CircleHelp",
	"XCircle":      "CircleX",
	"PlusCircle":   "CirclePlus",
	"MinusCircle":  "CircleMinus",
	"Loader2":      "LoaderCircle",
}

func loadLucide() {
	lucideSrc = map[string]string{}
	zr, err := gzip.NewReader(bytes.NewReader(lucideGz))
	if err != nil {
		return
	}
	raw, err := io.ReadAll(zr)
	if err != nil {
		return
	}
	_ = json.Unmarshal(raw, &lucideSrc)
}

// pascal is the viewer's kebab-case → PascalCase icon name conversion.
func pascal(name string) string {
	var b strings.Builder
	up := true
	for _, ch := range name {
		if ch == '-' {
			up = true
			continue
		}
		if up && ch >= 'a' && ch <= 'z' {
			ch -= 'a' - 'A'
		}
		up = false
		b.WriteRune(ch)
	}
	return b.String()
}

// lucideIcon returns the parsed icon for a kebab-case (or PascalCase)
// Lucide name, and whether it exists. A missing icon falls back to the
// neutral circle-help, as the viewer does.
func lucideIcon(name string) (*ui.SVG, bool) {
	lucideOnce.Do(loadLucide)
	key := pascal(strings.TrimSpace(name))
	if a, ok := lucideAliases[key]; ok {
		key = a
	}
	_, ok := lucideSrc[key]
	if !ok {
		key = "CircleHelp"
	}
	lucideMu.Lock()
	defer lucideMu.Unlock()
	if s := lucideCache[key]; s != nil {
		return s, ok
	}
	body := lucideSrc[key]
	src := `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">` + body + `</svg>`
	s, err := ui.ParseSVG([]byte(src))
	if err != nil {
		return nil, false
	}
	lucideCache[key] = s
	return s, ok
}

// icon builds a Lucide icon at size in col.
func (r *Render) icon(name string, size float32, col ui.Color) ui.Element {
	s, _ := lucideIcon(name)
	if s == nil {
		return ui.Box(r.C).Size(size, size).Shrink(0)
	}
	return ui.Icon(r.C, s).Size(size, size).Shrink(0).TextColor(col)
}
