package markdown

import "github.com/egoist/mygo/ui"

// Mathematical source remains lossless in span-only views and while the
// offline renderer is loading/unavailable. Unicode substitutions are not a
// layout engine and are deliberately no longer used.
func appendFormula(formula mathFormula, st Style, out *[]ui.Span) {
	if st.maths != nil {
		st.maths[len(*out)] = formula
	}
	appendPlainSpan(formula.Raw, st, out)
}

func appendPlainSpan(text string, st Style, out *[]ui.Span, mutators ...spanMutator) {
	if text == "" {
		return
	}
	span := ui.Span{Text: text, Size: st.FontSize, Color: st.FG}
	for _, m := range mutators {
		m(&span)
	}
	*out = append(*out, span)
}

func hasMath(formulas map[int]mathFormula, offset, count int) bool {
	for index := range formulas {
		if index >= offset && index < offset+count {
			return true
		}
	}
	return false
}
