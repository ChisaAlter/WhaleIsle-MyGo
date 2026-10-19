package text

import (
	"math"
	"testing"
)

func TestExactLineHeightKeepsFrameAndCentersTallRuns(t *testing.T) {
	s := Shared()
	p := Params{Text: "ABC鲸屿\nxyz设置", Style: Style{Family: "Segoe UI, Microsoft YaHei, sans-serif", Size: 24, LineHeight: .5}}
	natural := s.Layout(p)
	p.Style.ExactLineHeight = true
	exact := s.Layout(p)
	if len(exact.Lines) != 2 || exact.Height != 24 {
		t.Fatalf("fixed 12 DIP lines: %+v", exact)
	}
	for i, l := range exact.Lines {
		if l.Height != 12 || l.Y != float32(i)*12 || l.Ascent+l.Descent <= l.Height {
			t.Fatalf("line %d did not retain its frame around taller runs: %+v", i, l)
		}
		want := l.Y + (l.Height-l.Ascent-l.Descent)/2 + l.Ascent
		if math.Abs(float64(l.Baseline-want)) > .001 {
			t.Errorf("line %d baseline %v, want centered %v", i, l.Baseline, want)
		}
		if natural.Lines[i].Height < natural.Lines[i].Ascent+natural.Lines[i].Descent {
			t.Errorf("ordinary LineHeight stopped expanding for tall runs")
		}
	}
	p.Style.ExactLineHeight = false
	if s.Layout(p) != natural || natural == exact {
		t.Fatal("exact and ordinary line spacing share a cached layout")
	}
}
