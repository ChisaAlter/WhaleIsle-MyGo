package whaleui

import (
	"math"
	"os"
	"testing"
	"time"

	"whaleisle/internal/openui/lang"
)

// TestFunctionsMatchViewer replays testdata/function-cases.json, recorded
// from the viewer's own JavaScript implementations by gen/fncases.mjs.
func TestFunctionsMatchViewer(t *testing.T) {
	raw, err := os.ReadFile("testdata/function-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	v, err := lang.ParseJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	doc := v.(*lang.Object)
	var seq []float64
	for _, r := range doc.Value("random").([]lang.Value) {
		seq = append(seq, r.(float64))
	}
	now, _ := time.Parse(time.RFC3339, doc.Value("now").(string))
	impls := Functions()
	covered := map[string]bool{}
	for i, c := range doc.Value("cases").([]lang.Value) {
		co := c.(*lang.Object)
		name := co.Value("fn").(string)
		covered[name] = true
		args := co.Value("args").(*lang.Object)
		k := 0
		env := &lang.FuncEnv{
			Random:   func() float64 { r := seq[k%len(seq)]; k++; return r },
			Now:      func() time.Time { return now },
			Location: time.UTC,
			Locale:   "en",
		}
		got, err := impls[name](args, env)
		if co.Has("error") {
			if err == nil {
				t.Errorf("case %d %s(%s): expected error %v, got %s", i, name, lang.EncodeJSON(args), co.Value("error"), lang.EncodeJSON(got))
			}
			continue
		}
		if err != nil {
			t.Errorf("case %d %s(%s): unexpected error %v", i, name, lang.EncodeJSON(args), err)
			continue
		}
		want := string(lang.EncodeJSON(co.Value("result")))
		if wf, ok := co.Value("result").(float64); ok {
			// Go's math.Pow and V8 may differ in the last unit of precision.
			if gf, ok := got.(float64); ok && math.Abs(gf-wf) <= 1e-12*math.Max(1, math.Abs(wf)) {
				continue
			}
		}
		if g := string(lang.EncodeJSON(got)); g != want {
			t.Errorf("case %d %s(%s):\n  got  %s\n  want %s", i, name, lang.EncodeJSON(args), g, want)
		}
	}
	for name := range impls {
		if !covered[name] {
			t.Errorf("no reference case for %s", name)
		}
	}
}
