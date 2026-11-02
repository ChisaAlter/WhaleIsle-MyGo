package theme

import (
	"embed"
	"fmt"
	"log"
	"sync"

	"github.com/egoist/mygo/ui"
)

// These SFNT faces are decoded from the installed ui-theme WOFF2 subsets,
// combined at the four CSS weights the native interface uses. The accompanying
// OFL licenses and sources.json record provenance and glyph-coverage checks.
//
//go:embed fonts/*-400.ttf fonts/*-500.ttf fonts/*-600.ttf fonts/*-700.ttf
var skinFontFiles embed.FS

var skinFonts struct {
	sync.Once
	err error
}

var skinFontReport sync.Once

// EnsureSkinFonts reports a loading failure once; callers never silently claim
// that the installed font was available when registration failed.
func EnsureSkinFonts() {
	if err := RegisterSkinFonts(); err != nil {
		skinFontReport.Do(func() { log.Printf("theme skin fonts unavailable: %v", err) })
	}
}

// RegisterSkinFonts registers app-owned faces once without installing system
// fonts or changing any user preferences. It is safe to call before drawing.
func RegisterSkinFonts() error {
	skinFonts.Do(func() {
		entries, err := skinFontFiles.ReadDir("fonts")
		if err != nil {
			skinFonts.err = err
			return
		}
		for _, entry := range entries {
			data, err := skinFontFiles.ReadFile("fonts/" + entry.Name())
			if err == nil {
				err = ui.RegisterFont(data, "")
			}
			if err != nil {
				skinFonts.err = fmt.Errorf("register installed skin font %s: %w", entry.Name(), err)
				return
			}
		}
	})
	return skinFonts.err
}
