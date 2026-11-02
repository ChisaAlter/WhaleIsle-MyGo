package pluginui

import (
	"math"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

func validMirrorTextFocus(f *mirrorTextFocus) bool {
	if f == nil || f.DocumentKey == "" || len(f.DocumentKey) > 128 || f.Focus == "" || len(f.Focus) > 128 || !utf8.ValidString(f.Value) || len(f.Value) > 64<<10 || f.Subject.Kind != "session" || f.Subject.SessionID == "" || f.Subject.Epoch == "" || f.Handle == "" || len(f.Handle) > 64 || len(f.Entry) > 8192 || f.Turn < 0 || f.Seq > 1<<53-1 {
		return false
	}
	boundary := func(n int) bool {
		if n < 0 || n > ui.UTF16Len(f.Value) {
			return false
		}
		byteIndex := ui.UTF16ByteOffset(f.Value, n)
		return ui.UTF16Len(f.Value[:byteIndex]) == n
	}
	if f.Selection.End < f.Selection.Start || !boundary(f.Selection.Start) || !boundary(f.Selection.End) {
		return false
	}
	if f.Marked != nil && (f.Marked.End < f.Marked.Start || !boundary(f.Marked.Start) || !boundary(f.Marked.End)) {
		return false
	}
	for _, n := range []float64{f.Caret.X, f.Caret.Y, f.Caret.W, f.Caret.H} {
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return false
		}
	}
	return f.Caret.W > 0 && f.Caret.H > 0 && f.Caret.W <= 2048 && f.Caret.H <= 1536
}
