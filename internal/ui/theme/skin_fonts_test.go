package theme

import (
	"fmt"
	"math"
	"runtime"
	"testing"

	"github.com/egoist/mygo/ui"
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

func TestSkinFontsShapeWithTheInstalledFaceAndFallback(t *testing.T) {
	if err := RegisterSkinFonts(); err != nil {
		t.Fatal(err)
	}
	for _, stem := range []string{"inter-normal", "source-serif-4-normal", "source-serif-4-italic"} {
		family := "Source Serif 4"
		if stem == "inter-normal" {
			family = "Inter"
		}
		for _, weight := range []int{400, 500, 600, 700} {
			name := fmt.Sprintf("fonts/%s-%d.ttf", stem, weight)
			data, err := skinFontFiles.ReadFile(name)
			if err != nil {
				t.Fatal(err)
			}
			face, err := sfnt.Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			for _, letter := range []rune{'M', 'é', 'Ā'} {
				index, err := face.GlyphIndex(nil, letter)
				if err != nil || index == 0 {
					t.Fatalf("%s lacks source glyph %U", name, letter)
				}
				advance, err := face.GlyphAdvance(nil, index, fixed.I(20), font.HintingNone)
				if err != nil {
					t.Fatal(err)
				}
				got, _ := ui.ShapeTextLine(string(letter), ui.Font{Family: family, Size: 20, Weight: weight, Italic: stem == "source-serif-4-italic"}, 0).Size()
				want := float32(advance) / 64
				if math.Abs(float64(got-want)) > 0.1 {
					t.Errorf("%s %U resolved another face: advance %g want installed %g", name, letter, got, want)
				}
			}
		}
	}
	for _, text := range []string{"Ω", "鲸", "😀"} {
		width, height := ui.ShapeTextLine(text, ui.Font{Family: ClaudeProseFamily, Size: 20}, 0).Size()
		if width <= 0 || height <= 0 {
			t.Errorf("fallback text %q has no layout", text)
		}
	}
	if runtime.GOOS == "windows" {
		for _, weight := range []int{400, 500, 600, 700} {
			want, _ := ui.ShapeTextLine("WhaleIsle 123", ui.Font{Family: "Inter", Size: 16, Weight: weight}, 0).Size()
			got, _ := ui.ShapeTextLine("WhaleIsle 123", ui.Font{Family: DefaultTextFamily, Size: 16, Weight: weight}, 0).Size()
			if math.Abs(float64(got-want)) > 0.1 {
				t.Errorf("Windows DefaultTextFamily weight=%d advance=%g want Inter %g", weight, got, want)
			}
		}
		for _, text := range []string{"鲸屿设置", "WhaleIsle 鲸屿", "😀"} {
			w, h := ui.ShapeTextLine(text, ui.Font{Family: DefaultTextFamily, Size: 14, Weight: 500}, 0).Size()
			if w <= 0 || h <= 0 {
				t.Errorf("Windows DefaultTextFamily fallback text %q has no layout", text)
			}
		}
	}
}
