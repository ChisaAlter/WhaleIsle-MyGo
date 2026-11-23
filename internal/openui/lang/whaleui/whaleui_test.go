package whaleui

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/openui/lang"
)

func mustLib(t testing.TB) *lang.Library {
	t.Helper()
	l, err := Library()
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func render(t testing.TB, program string) (*lang.Instance, *lang.Frame) {
	t.Helper()
	in := lang.NewInstance(mustLib(t), lang.Options{Seed: 7, Now: func() time.Time { return time.Date(2026, 11, 23, 10, 30, 0, 0, time.UTC) }, Location: time.UTC})
	return in, in.Update(lang.Update{Program: program})
}

func errorsOf(f *lang.Frame) []string {
	var out []string
	for _, d := range f.Errors {
		if !d.Warning() {
			out = append(out, d.Code+": "+d.Message)
		}
	}
	return out
}

func TestLibraryStats(t *testing.T) {
	l := mustLib(t)
	if len(l.Components) != 73 || len(l.Functions) != 59 || len(l.Actions) != 1 || l.Root != "Col" {
		t.Fatalf("components=%d functions=%d actions=%d root=%q", len(l.Components), len(l.Functions), len(l.Actions), l.Root)
	}
	g, _ := GenuiValidator()
	if len(g.Schemas) != 50 || len(g.Types) != 15 {
		t.Fatalf("genui schemas=%d types=%d", len(g.Schemas), len(g.Types))
	}
	for _, name := range []string{"Input", "TextArea", "Select", "SegmentedControl", "Slider", "Checkbox", "RadioGroup", "DatePicker", "Timer", "Map"} {
		bound := false
		for _, p := range l.Components[name].Params {
			bound = bound || p.Bindable
		}
		if !bound {
			t.Errorf("%s has no bindable param", name)
		}
	}
	for name, f := range l.Functions {
		if f.Impl == nil {
			t.Errorf("function %s has no implementation", name)
		}
	}
	for _, name := range []string{"Pick", "Shuffle", "Random", "Now", "Today"} {
		if l.Functions[name].Pure {
			t.Errorf("%s should be impure", name)
		}
	}
}

// TestGeneratedLibraryMatchesViewer fails when the bundled viewer changed
// without regenerating library.json (node gen/gen.mjs).
func TestGeneratedLibraryMatchesViewer(t *testing.T) {
	host, err := os.ReadFile(filepath.Join("..", "..", "..", "desktop", "openui", "viewer", "v2", "host.cjs"))
	if err != nil {
		t.Skip("viewer host.cjs not present:", err)
	}
	sum := sha256.Sum256(host)
	var doc struct {
		Sources struct {
			Host struct {
				SHA256 string `json:"sha256"`
			} `json:"host"`
		} `json:"sources"`
	}
	if err := json.Unmarshal(libraryJSON, &doc); err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(sum[:]); got != doc.Sources.Host.SHA256 {
		t.Fatalf("host.cjs sha256 %s differs from generated %s: run `go generate ./internal/openui/lang/whaleui`", got, doc.Sources.Host.SHA256)
	}
}

func TestGuideExamplesRender(t *testing.T) {
	raw, err := os.ReadFile("testdata/guide-examples.json")
	if err != nil {
		t.Fatal(err)
	}
	var examples []string
	if err := json.Unmarshal(raw, &examples); err != nil {
		t.Fatal(err)
	}
	for i, ex := range examples {
		_, f := render(t, ex)
		if f.Root == nil || len(errorsOf(f)) > 0 {
			t.Errorf("example %d: root=%v errors=%v", i, f.Root != nil, errorsOf(f))
		}
	}
}

func TestEstimateExampleInteraction(t *testing.T) {
	prog := `root = Col([Title("Estimate","md"), Text(@Format(total,"currency",2,"USD")), Input($quantity,"Quantity","number"), Button("Reset",@Reset($quantity),"outline")])
$quantity = 1
$price = 12
qty = @Clamp(@Default($quantity,0),0,1000)
total = qty * $price`
	in, f := render(t, prog)
	if errs := errorsOf(f); len(errs) > 0 {
		t.Fatal(errs)
	}
	text := f.Root.Children()[1].Prop("text")
	if text != "$12.00" {
		t.Fatalf("total text = %v", text)
	}
	b := f.Root.Children()[2].Prop("value").(*lang.Binding)
	if err := in.SetBinding(b, 3.0); err != nil {
		t.Fatal(err)
	}
	f = in.Render()
	if got := f.Root.Children()[1].Prop("text"); got != "$36.00" {
		t.Fatalf("after input: %v", got)
	}
	run := in.Dispatch("Reset", "", f.Root.Children()[3].Prop("action"))
	if eff, err := run.Next(); eff != nil || err != nil {
		t.Fatalf("reset should be local: %v %v", eff, err)
	}
	if got := in.Render().Root.Children()[1].Prop("text"); got != "$12.00" {
		t.Fatalf("after reset: %v", got)
	}
}

func TestWhaleProfileDiagnostics(t *testing.T) {
	_, f := render(t, "root = Col([Text(x)])\n$a = [1, b]\nb = 2")
	codes := map[string]bool{}
	for _, d := range f.Errors {
		codes[d.Code] = true
	}
	if !codes["unresolved-reference"] || !codes["invalid-state-default"] {
		t.Fatalf("codes = %v", codes)
	}
	_, f = render(t, `root = Col([Text("a")])`+"\nq = Query(\"t\", {}, {})")
	found := false
	for _, d := range f.Errors {
		found = found || d.Code == lang.CodeDataUnavailable
	}
	if !found {
		t.Fatal("Query must be rejected without a business-tool provider")
	}
}

func TestUnicodeIdentifiers(t *testing.T) {
	_, f := render(t, "root = Col([Text(标题)])\n标题 = \"你好\"")
	if errs := errorsOf(f); len(errs) > 0 || f.Root.Children()[0].Prop("text") != "你好" {
		t.Fatalf("errors=%v", errs)
	}
}

func TestSpecializedComponentValidation(t *testing.T) {
	valid := map[string]string{
		"Plot":      `{series: [{expr: "sin(x)"}], xMin: -6, xMax: 6}`,
		"Mermaid":   `{code: "graph TD; A-->B"}`,
		"Scene3D":   `{meshes: [{shape: "box"}]}`,
		"EChart":    `{preset: "bar", data: [{name: "a", value: 1}]}`,
		"Diagram":   `{kind: "flowchart", nodes: [{id: "a", label: "A"}], edges: [{from: "a", to: "a"}]}`,
		"Blocks":    `{rows: [{cells: ["a", "b"]}]}`,
		"Quiz":      `{question: "2+2?", options: ["3", "4"]}`,
		"FileTree":  `{items: [{name: "src", children: [{name: "a.go"}]}]}`,
		"Timeline":  `{items: [{title: "Start", time: "09:00"}]}`,
		"Steps":     `{steps: [{title: "One"}, {title: "Two"}], current: 1}`,
		"KeyValue":  `{pairs: [["a", "1"], ["b", "2"]]}`,
		"Diff":      `{diffs: [{path: "a.txt", oldText: "a", newText: "b"}]}`,
		"Json":      `{value: {a: 1}}`,
		"Tabs":      `{tabs: [{label: "A", items: [{type: "text", content: "x"}]}]}`,
		"Accordion": `{items: [{title: "A", items: [{type: "text", content: "x"}]}]}`,
	}
	invalid := map[string]string{
		"Plot":      `{xMin: 0}`,
		"Mermaid":   `{code: 3}`,
		"Scene3D":   `{meshes: [{shape: "pyramid"}]}`,
		"EChart":    `{preset: "bar"}`,
		"Diagram":   `{kind: "nope", nodes: []}`,
		"Blocks":    `{rows: [{label: "x"}]}`,
		"Quiz":      `{question: "q"}`,
		"FileTree":  `{items: [{type: "file"}]}`,
		"Timeline":  `{items: [{time: "x"}]}`,
		"Steps":     `{steps: "one"}`,
		"KeyValue":  `{pairs: [{key: "a"}]}`,
		"Diff":      `{diffs: [{path: "a"}]}`,
		"Json":      `{}`,
		"Tabs":      `{tabs: [{items: []}]}`,
		"Accordion": `{items: [{title: "A"}]}`,
	}
	for name, opts := range valid {
		_, f := render(t, "root = Col([c])\nc = "+name+"("+opts+")")
		if errs := errorsOf(f); len(errs) > 0 || len(f.Root.Children()[0].Issues) > 0 {
			t.Errorf("%s valid options rejected: %v", name, errs)
		}
	}
	for name, opts := range invalid {
		_, f := render(t, "root = Col([c])\nc = "+name+"("+opts+")")
		el := f.Root.Children()[0]
		if len(el.Issues) == 0 {
			t.Errorf("%s invalid options accepted", name)
			continue
		}
		found := false
		for _, d := range f.Errors {
			found = found || d.Code == lang.CodeSchemaInvalid && d.StatementID == "c"
		}
		if !found {
			t.Errorf("%s: no schema-invalid diagnostic", name)
		}
	}
	_, f := render(t, `root = Col([Genui({items: [{type: "text", content: "hi"}, {type: "nope"}]})])`)
	if issues := f.Root.Children()[0].Issues; len(issues) != 1 || !strings.Contains(issues[0], "unsupported component type 'nope'") {
		t.Fatalf("genui issues = %v", issues)
	}
	// Aliases normalize before validation: callout {text} is content.
	g, _ := GenuiValidator()
	if issues := g.ValidateSpec(lang.ObjectFrom("type", "callout", "text", "hi")); len(issues) != 0 {
		t.Fatalf("alias not applied: %v", issues)
	}
}

func TestPrepareProgram(t *testing.T) {
	p := PrepareProgram("dsh-ui", `{"items":[{"type":"text","content":"hi"}],"panel":true}`, false)
	if !p.Panel || !strings.Contains(p.Program, `genui = Genui({"items":[{"type":"text","content":"hi"}]})`) {
		t.Fatalf("%+v", p)
	}
	_, f := render(t, p.Program)
	if errs := errorsOf(f); len(errs) > 0 {
		t.Fatal(errs)
	}
	p = PrepareProgram("dsh-ui", `{"items":[{"type":"text","content":"h`, true)
	if !strings.Contains(p.Program, `"content":"h"`) {
		t.Fatalf("partial spec: %q", p.Program)
	}
	if p := PrepareProgram("openui", "root = Col([])", false); p.Program != "root = Col([])" {
		t.Fatal(p)
	}
	if p := PrepareProgram("bogus", "root = Col([])", false); len(p.Diagnostics) == 0 {
		t.Fatal("unsupported language accepted")
	}
}

func TestCopyToClipboardEffect(t *testing.T) {
	in, f := render(t, `root = Col([Button("Copy", Action([@Set($n, $n + 1), @CopyToClipboard("v" + $n)]))])`+"\n$n = 1")
	run := in.Dispatch("Copy", "", f.Root.Children()[0].Prop("action"))
	eff, err := run.Next()
	if err != nil || eff == nil || eff.Kind != lang.EffectCopyToClipboard || eff.Text != "v2" {
		t.Fatalf("eff=%+v err=%v", eff, err)
	}
	if eff, err := run.Next(); eff != nil || err != nil {
		t.Fatal(eff, err)
	}
}
