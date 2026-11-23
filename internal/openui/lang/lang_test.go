package lang

import (
	"encoding/json"
	"math"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func newFixtureInstance(t testing.TB) *Instance {
	return NewInstance(loadFixtureLibrary(t), Options{})
}

func codes(ds []Diagnostic) map[string]int {
	m := map[string]int{}
	for _, d := range ds {
		m[d.Code]++
	}
	return m
}

func TestNumberToString(t *testing.T) {
	cases := map[float64]string{
		0: "0", 1: "1", -1.5: "-1.5", 0.1: "0.1", 1e21: "1e+21", 1e20: "100000000000000000000",
		123456789012345680000: "123456789012345680000", 1e-7: "1e-7", 0.000001: "0.000001", 1.5e-10: "1.5e-10",
		math.Inf(1): "Infinity", math.NaN(): "NaN", 2.5e25: "2.5e+25",
	}
	for in, want := range cases {
		if got := NumberToString(in); got != want {
			t.Errorf("NumberToString(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestNumberToStringRuntime(t *testing.T) {
	a, b := 0.1, 0.2
	if got := NumberToString(a + b); got != "0.30000000000000004" {
		t.Fatal(got)
	}
}

func TestStringToNumber(t *testing.T) {
	cases := map[string]float64{"": 0, "  42 ": 42, "0x1F": 31, "1e3": 1000, ".5": 0.5, "5.": 5, "-Infinity": math.Inf(-1), "+3": 3, "0b101": 5}
	for in, want := range cases {
		if got := StringToNumber(in); got != want {
			t.Errorf("StringToNumber(%q) = %v, want %v", in, got, want)
		}
	}
	for _, bad := range []string{"abc", "1_000", "0x", "1e", "--1", "inf", "NaN"} {
		if !math.IsNaN(StringToNumber(bad)) {
			t.Errorf("StringToNumber(%q) should be NaN", bad)
		}
	}
	if ParseFloat("12abc") != 12 || !math.IsNaN(ParseFloat("abc")) {
		t.Error("ParseFloat prefix semantics")
	}
}

func TestLooseEquality(t *testing.T) {
	yes := [][2]Value{{5.0, "5"}, {"", 0.0}, {nil, nil}, {true, 1.0}, {[]Value{1.0, 2.0}, "1,2"}, {"0", false}}
	no := [][2]Value{{nil, 0.0}, {nil, ""}, {"abc", 0.0}, {math.NaN(), math.NaN()}, {[]Value{1.0}, []Value{1.0}}}
	for _, c := range yes {
		if !LooseEqual(c[0], c[1]) {
			t.Errorf("%v == %v should be true", c[0], c[1])
		}
	}
	for _, c := range no {
		if LooseEqual(c[0], c[1]) {
			t.Errorf("%v == %v should be false", c[0], c[1])
		}
	}
}

func TestObjectKeyOrder(t *testing.T) {
	o := NewObject(0)
	o.Set("b", 1.0)
	o.Set("10", 1.0)
	o.Set("a", 1.0)
	o.Set("2", 1.0)
	if got := strings.Join(o.Keys(), ","); got != "2,10,b,a" {
		t.Fatalf("keys = %s", got)
	}
}

// evalExpr evaluates one expression through the fixture library.
func evalExpr(t testing.TB, expr string, extra ...string) Value {
	t.Helper()
	in := newFixtureInstance(t)
	prog := "root = Stack([Col(\"x\", [v])])\nv = " + expr + "\n" + strings.Join(extra, "\n")
	f := in.Update(Update{Program: prog})
	if f.Root == nil {
		t.Fatalf("%s: no root, errors %+v", expr, f.Errors)
	}
	col := f.Root.Children()[0]
	data, _ := col.Prop("data").([]Value)
	if len(data) != 1 {
		t.Fatalf("%s: data %v errors %+v", expr, data, f.Errors)
	}
	return data[0]
}

func TestBuiltins(t *testing.T) {
	rows := `rows = [{n: 3, s: "b", c: {name: "x"}}, {n: 1, s: "a", c: {name: "y"}}, {n: 2, s: "B", c: {name: "z"}}]`
	cases := []struct {
		expr string
		want string
	}{
		{"@Count([1, 2, 3])", "3"}, {"@Count(5)", "0"},
		{"@First([7, 8])", "7"}, {"@First([])", "null"}, {"@Last([7, 8])", "8"}, {"@Last(3)", "null"},
		{`@Sum([1, "2", true, null])`, "4"}, {"@Sum(1)", "0"},
		{"@Avg([1, 2, 6])", "3"}, {"@Avg([])", "0"},
		{`@Min([3, "1", 2])`, "1"}, {"@Max([3, 9, 2])", "9"}, {"@Min([])", "0"},
		{`@Filter(rows, "n", ">", 1)`, `[{"n":3,"s":"b","c":{"name":"x"}},{"n":2,"s":"B","c":{"name":"z"}}]`},
		{`@Filter(rows, "s", "contains", "B")`, `[{"n":2,"s":"B","c":{"name":"z"}}]`},
		{`@Filter(rows, "c.name", "==", "y")`, `[{"n":1,"s":"a","c":{"name":"y"}}]`},
		{`@Filter([1, 2, 3], "", "!=", 2)`, "[1,3]"}, {`@Filter(rows, "n", "~", 1)`, "[]"}, {`@Filter(1, "", "==", 1)`, "[]"},
		{`@Sort(rows, "s").s`, `["a","b","B"]`}, {`@Sort(rows, "n", "desc").n`, "[3,2,1]"}, {`@Sort([10, "9", 1], "")`, `[1,"9",10]`},
		{"@Round(2.5)", "3"}, {"@Round(-2.5)", "-2"}, {"@Round(1.005, 2)", "1"}, {"@Round(3.14159, 2)", "3.14"},
		{`@Abs(-3)`, "3"}, {`@Floor("2.7")`, "2"}, {"@Ceil(2.1)", "3"},
		{`@Each([1, 2], "x", x * 10)`, "[10,20]"}, {`@Each(5, "x", x)`, "[]"},
		{`@Each([[1, 2], [3]], "x", @Each(x, "x", x + 1))`, "[[2,3],[4]]"},
	}
	for _, c := range cases {
		got := string(EncodeJSON(evalExpr(t, c.expr, rows)))
		if got != c.want {
			t.Errorf("%s = %s, want %s", c.expr, got, c.want)
		}
	}
}

func TestExpressions(t *testing.T) {
	cases := []struct{ expr, want string }{
		{`"a" + 1 + 2`, `"a12"`}, {`1 + 2 + "a"`, `"3a"`}, {`"x" + null`, `"x"`}, {`"v" + [1, [2, 3]]`, `"v1,2,3"`}, {`"o" + {a: 1}`, `"o[object Object]"`},
		{`10 % 4`, "2"}, {`-7 % 3`, "-1"}, {`1 / 0`, "0"}, {`"3" * "4"`, "12"},
		{`true ? "a" : "b"`, `"a"`}, {`0 ? "a" : "" ? "b" : "c"`, `"c"`},
		{`[{a: 1}, {b: 2}].a`, "[1,null]"}, {`[1, 2, 3].length`, "3"}, {`"héllo".length`, "5"}, {`"abc"[1]`, `"b"`},
		{`{a: {b: [5, 6]}}.a.b[1]`, "6"}, {`[1, 2][5]`, "null"}, {`[1, 2][0.5]`, "null"}, {`null.x`, "null"}, {`{k: 1}["k"]`, "1"},
		{`!!"x" && 0`, "0"}, {`null || "d"`, `"d"`}, {`-"5"`, "-5"}, {`!0`, "true"}, {`2 > 1 == true`, "true"}, {`"10" < "9"`, "false"},
		{`{"quoted key": 1, $s: 2, Card: 3, 4: 4}`, `{"4":4,"quoted key":1,"s":2,"Card":3}`},
		{`$missing == null`, "true"},
	}
	for _, c := range cases {
		got := string(EncodeJSON(evalExpr(t, c.expr)))
		if got != c.want {
			t.Errorf("%s = %s, want %s", c.expr, got, c.want)
		}
	}
}

// TestStreamingByteByByte checks "streaming equals parsing" (spec section 4)
// for every program fixture streamed one byte at a time.
func TestStreamingByteByByte(t *testing.T) {
	lib := loadFixtureLibrary(t)
	for _, id := range fixtureCases(t) {
		files := readCase(t, filepath.Join(fixturesDir, id))
		text, ok := files["input.oui"]
		if !ok {
			if c, ok := files["chunks.json"]; ok {
				var chunks []string
				_ = json.Unmarshal([]byte(c), &chunks)
				text = strings.Join(chunks, "")
			} else {
				continue
			}
		}
		batch := NewInstance(lib, Options{})
		want := evaluateCase(batch, batch.Update(Update{Program: text}))
		in := NewInstance(lib, Options{})
		var f *Frame
		for i := 1; i <= len(text); i++ {
			f = in.Update(Update{Program: text[:i], Streaming: i < len(text)})
			if f.Streaming && len(f.Errors) > 0 {
				for _, d := range f.Errors {
					if d.Code != CodeRuntimeError {
						t.Errorf("%s: parser error %s while streaming at %d", id, d.Code, i)
					}
				}
			}
		}
		got := evaluateCase(in, f)
		if !reflect.DeepEqual(got.root, want.root) || !reflect.DeepEqual(got.state, want.state) || got.incomplete != want.incomplete ||
			!reflect.DeepEqual(asSet(got.unresolved), asSet(want.unresolved)) || !reflect.DeepEqual(asSet(got.orphans), asSet(want.orphans)) || len(got.errors) != len(want.errors) {
			t.Errorf("%s: streamed %s / %v, batch %s / %v", id, js(got.root), got.errors, js(want.root), want.errors)
		}
	}
}

func TestStreamingEdgeCases(t *testing.T) {
	lib := loadFixtureLibrary(t)
	sp := NewStreamParser(lib, Limits{})

	// Root first: the shell renders before its children arrive, no errors.
	r := sp.Set("root = Stack([a, b])\na = Text(\"A\")\nb = Te", false)
	if r.Root == nil || len(r.Root.Props.Value("children").([]Value)) != 1 {
		t.Fatalf("root-first partial: %+v", r.Root)
	}
	// Missing root while streaming is not fatal in the frame.
	in := NewInstance(lib, Options{})
	f := in.Update(Update{Program: "a = Text(\"x\")\n", Streaming: true})
	if codes(f.Errors)["no-root"] != 0 {
		t.Fatal("no-root reported while streaming")
	}
	if f = in.Update(Update{Program: "a = Text(\"x\")\n"}); codes(f.Errors)["no-root"] != 1 {
		t.Fatal("no-root missing at stream end")
	}

	// A multi-line ternary does not complete before its `:` line arrives.
	sp = NewStreamParser(lib, Limits{})
	sp.Set("root = Stack([Text(s)])\ns = 1 > 0\n", false)
	r = sp.Set("root = Stack([Text(s)])\ns = 1 > 0\n  ? \"yes\"\n", false)
	if got := r.Root.Props.Value("children").([]Value); len(got) != 1 {
		t.Fatalf("ternary children %v", got)
	}
	r = sp.Set("root = Stack([Text(s)])\ns = 1 > 0\n  ? \"yes\"\n  : \"no\"\n", true)
	in = NewInstance(lib, Options{})
	f = in.Update(Update{Program: "root = Stack([Text(s)])\ns = 1 > 0\n  ? \"yes\"\n  : \"no\"\n"})
	if f.Root.Children()[0].Prop("text") != "yes" {
		t.Fatal("ternary value")
	}

	// An unterminated string closes implicitly and marks the result incomplete.
	r = Parse("root = Stack([Text(\"open", lib, Limits{})
	if !r.Incomplete || r.Root == nil {
		t.Fatal("unterminated string")
	}

	// A pending statement that needed closing does not replace a completed one.
	sp = NewStreamParser(lib, Limits{})
	r = sp.Set("root = Stack([t])\nt = Text(\"done\")\nt = Text(\"pen", false)
	if txt := r.Root.Props.Value("children").([]Value)[0].(*Element).Props.Value("text"); txt != "done" {
		t.Fatalf("pending overwrote completed: %v", txt)
	}
	r = sp.Set("root = Stack([t])\nt = Text(\"done\")\nt = Text(\"pend\")", true)
	if txt := r.Root.Props.Value("children").([]Value)[0].(*Element).Props.Value("text"); txt != "pend" {
		t.Fatalf("final redefinition must win: %v", txt)
	}

	// A $name at the very end may still grow: it is not auto-declared yet.
	sp = NewStreamParser(lib, Limits{})
	r = sp.Set("root = Stack([Text($na", false)
	if r.StateDefaults.Has("$na") {
		t.Fatal("partial $name declared")
	}
	if r = sp.Set("root = Stack([Text($na", true); !r.StateDefaults.Has("$na") {
		t.Fatal("complete $name not declared")
	}

	// A fence opener arriving late resets the cache (reconciliation).
	sp = NewStreamParser(lib, Limits{})
	sp.Set("x = 1\nroot = Stack([])\n", false)
	r = sp.Set("x = 1\nroot = Stack([])\nNow in a fence:\n```openui\nroot = Stack([Text(\"f\")])\n", false)
	if len(r.Statements) != 1 || r.Root == nil {
		t.Fatalf("fence reset: %d statements", len(r.Statements))
	}

	// A user-changed state survives a later parse; a pristine default is replaced.
	in = NewInstance(lib, Options{})
	in.Update(Update{Program: "$a = 1\n$b = 1\nroot = Stack([])", Streaming: true})
	_ = in.SetState("$a", 5.0)
	in.Update(Update{Program: "$a = 2\n$b = 2\nroot = Stack([])"})
	if in.Store().Get("$a") != 5.0 || in.Store().Get("$b") != 2.0 {
		t.Fatalf("state: a=%v b=%v", in.Store().Get("$a"), in.Store().Get("$b"))
	}
}

func TestRevisions(t *testing.T) {
	in := newFixtureInstance(t)
	in.Update(Update{Revision: 3, Program: "root = Stack([Text(\"three\")])"})
	f := in.Update(Update{Revision: 2, Program: "root = Stack([Text(\"two\")])"})
	if codes(f.Errors)[CodeStaleRevision] != 1 || f.Root.Children()[0].Prop("text") != "three" || f.Revision != 3 {
		t.Fatal("stale revision applied")
	}
	f = in.Update(Update{Revision: 5, Program: "root = Stack([Text(\"five\")])"})
	if f.Root.Children()[0].Prop("text") != "five" {
		t.Fatal("skipped revision rejected")
	}
}

func TestLimits(t *testing.T) {
	lib := loadFixtureLibrary(t)
	deep := "root = Stack([Text(" + strings.Repeat("(", 500) + "\"x\"" + strings.Repeat(")", 500) + ")])"
	if r := Parse(deep, lib, Limits{}); codes(r.Errors)[CodeBudgetExceeded] != 1 || !r.Fatal {
		t.Fatalf("depth: %+v", r.Errors)
	}
	big := "root = Stack([])\n" + strings.Repeat("x = 1\n", 60000)
	in := NewInstance(lib, Options{})
	if f := in.Update(Update{Program: big}); codes(f.Errors)[CodeProgramTooLarge] != 1 {
		t.Fatalf("program size: %+v", f.Errors)
	}
	if r := Parse(strings.Repeat("a = [1, 2, 3, 4, 5, 6, 7, 8]\n", 6000), lib, Limits{}); codes(r.Errors)[CodeBudgetExceeded] != 1 {
		t.Fatal("token budget")
	}
	if r := Parse(strings.Repeat("a = 1\n", 10), lib, Limits{MaxStatements: 5}); codes(r.Errors)[CodeBudgetExceeded] != 0 {
		t.Fatal("duplicate names count once")
	}
	var sb strings.Builder
	for i := range 10 {
		sb.WriteString("s" + itoa(i) + " = 1\n")
	}
	if r := Parse(sb.String(), lib, Limits{MaxStatements: 5}); codes(r.Errors)[CodeBudgetExceeded] != 1 {
		t.Fatal("statement budget")
	}
	// Evaluation budget: nested @Each over large ranges.
	in = NewInstance(lib, Options{})
	f := in.Update(Update{Program: "root = Stack([Col(\"x\", @Each(a, \"i\", @Each(a, \"j\", i * j)))])\na = [" + strings.Repeat("1,", 400) + "1]"})
	if codes(f.Errors)[CodeBudgetExceeded] != 1 || f.Root != nil {
		t.Fatalf("eval budget: %+v", f.Errors)
	}
	// Render node budget.
	in = NewInstance(lib, Options{Limits: Limits{MaxRenderNodes: 50}})
	f = in.Update(Update{Program: "root = Stack(@Each(a, \"i\", Text(\"t\")))\na = [" + strings.Repeat("1,", 100) + "1]"})
	if codes(f.Errors)[CodeBudgetExceeded] != 1 {
		t.Fatalf("render budget: %+v", f.Errors)
	}
	// State key budget.
	in = NewInstance(lib, Options{Limits: Limits{MaxStateKeys: 3}})
	in.Update(Update{Program: "root = Stack([])\n$a = 1\n$b = 2"})
	_ = in.SetState("$c", 1.0)
	if err := in.SetState("$d", 1.0); err == nil {
		t.Fatal("state key budget")
	}
}

func clickPlan(t *testing.T, in *Instance, f *Frame, id string) *ActionRun {
	t.Helper()
	el := findElement(f.Root, id)
	if el == nil {
		t.Fatalf("no element %s", id)
	}
	label, _ := el.Prop("label").(string)
	return in.Dispatch(label, el.FormName, el.Prop("action"))
}

func TestActions(t *testing.T) {
	in := newFixtureInstance(t)
	prog := `root = Stack([bad, cred, ok, ctx, plain, legacy, stop])
bad = Button("Bad", @OpenUrl("javascript:alert(1)"))
cred = Button("Cred", @OpenUrl("https://user:pw@example.com/"))
ok = Button("Ok", @OpenUrl("https://example.com/a?b=1"))
ctx = Button("Ctx", @ToAssistant("Hi " + $name, {id: 7}))
plain = Button("Plain")
legacy = Button("Legacy", {type: "open_url", url: "https://openui.com"})
stop = Button("Stop", Action([@Set($n, 1), @OpenUrl("ftp://x"), @Set($n, 2)]))
$name = "Ann"`
	f := in.Update(Update{Program: prog})
	for _, id := range []string{"bad", "cred"} {
		if eff, err := clickPlan(t, in, f, id).Next(); err == nil || eff != nil {
			t.Errorf("%s: invalid URL accepted", id)
		}
	}
	eff, err := clickPlan(t, in, f, "ok").Next()
	if err != nil || eff.Kind != EffectOpenURL || eff.URL != "https://example.com/a?b=1" {
		t.Fatalf("ok url: %+v %v", eff, err)
	}
	_ = in.SetState("$name", "Bo")
	eff, _ = clickPlan(t, in, f, "ctx").Next()
	if eff.Event.HumanFriendlyMessage != "Hi Bo" || string(EncodeJSON(eff.Event.Params)) != `{"context":{"id":7}}` {
		t.Fatalf("deferred args: %+v", eff.Event)
	}
	if eff, _ = clickPlan(t, in, f, "plain").Next(); eff.Kind != EffectToAssistant || eff.Event.HumanFriendlyMessage != "Plain" {
		t.Fatal("default action")
	}
	if eff, _ = clickPlan(t, in, f, "legacy").Next(); eff.Kind != EffectOpenURL {
		t.Fatal("legacy action")
	}
	run := clickPlan(t, in, f, "stop")
	if eff, err := run.Next(); eff != nil || err == nil {
		t.Fatal("failed step must stop")
	}
	if in.Store().Get("$n") != 1.0 {
		t.Fatalf("$n = %v", in.Store().Get("$n"))
	}
	// Host failure stops the rest.
	f = in.Update(Update{Program: "root = Stack([b])\nb = Button(\"B\", Action([@ToAssistant(\"a\"), @Set($z, 1)]))"})
	run = clickPlan(t, in, f, "b")
	if eff, _ := run.Next(); eff == nil {
		t.Fatal("expected effect")
	}
	run.Fail(nil)
	if eff, err := run.Next(); eff != nil || err == nil || in.Store().Get("$z") != nil {
		t.Fatal("Fail must stop the plan")
	}
	// Actions do nothing while streaming.
	in.Update(Update{Program: "root = Stack([b])\nb = Button(\"B\")", Streaming: true})
	if eff, _ := in.Dispatch("B", "", nil).Next(); eff != nil {
		t.Fatal("action ran while streaming")
	}
}

func TestPersistence(t *testing.T) {
	lib := loadFixtureLibrary(t)
	in := NewInstance(lib, Options{})
	f := in.Update(Update{Program: "$q = \"\"\n$keep = 1\nroot = Stack([Form(\"f\", [i]), p])\ni = Input(\"email\", \"E\")\np = Input(\"note\", \"N\")"})
	_ = in.SetState("$q", "typed")
	_ = in.SetField(findElement(f.Root, "i").FormName, "Input", "email", "a@b.co")
	_ = in.SetField("", "Input", "note", "n")
	raw, err := in.PersistState()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "$keep") || !strings.Contains(string(raw), `"format":"whale-ui-state"`) || !strings.Contains(string(raw), `"__whaleField":1`) {
		t.Fatalf("persisted %s", raw)
	}
	in2 := NewInstance(lib, Options{})
	if err := in2.RestoreState(raw); err != nil {
		t.Fatal(err)
	}
	in2.Update(Update{Program: "$q = \"\"\nroot = Stack([])"})
	if in2.Store().Get("$q") != "typed" || in2.FieldValue("f", "email") != "a@b.co" || in2.FieldValue("", "note") != "n" {
		t.Fatalf("restore: %s", EncodeJSON(in2.Store().Snapshot()))
	}
	// Legacy flat documents.
	legacy := []byte(`{"$x": 2, "f": {"a": {"value": "v", "componentType": "Input"}}, "g": {"value": 1, "componentType": "Input"}}`)
	vals, err := UnmarshalState(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := vals.Value("g").(*Field); !ok {
		t.Fatal("legacy field")
	}
	if fo, _ := vals.Value("f").(*Object); fo == nil || fo.Value("a").(*Field).Value != "v" {
		t.Fatal("legacy form field")
	}
	// The 48 KiB bound.
	_ = in.SetState("$big", strings.Repeat("x", 50<<10))
	if _, err := in.PersistState(); err == nil || !strings.Contains(err.Error(), CodeStateTooLarge) {
		t.Fatalf("state bound: %v", err)
	}
	if _, err := UnmarshalState([]byte(`{"format":"whale-ui-state","version":2,"values":{}}`)); err == nil {
		t.Fatal("unsupported version accepted")
	}
}

func TestMergeAndSerialize(t *testing.T) {
	lib := loadFixtureLibrary(t)
	got := MergeStatements("root = Stack([a])\na = Text(\"A\")\ns = $x\n  ? 1\n  : 2", "s = 3", lib)
	if got != "root = Stack([a])\na = Text(\"A\")" {
		t.Fatalf("merge: %q", got)
	}
	in := NewInstance(lib, Options{})
	f := in.Update(Update{Program: `root = Stack([Text("a - (b - c)"), Col("x", [1, -2])], "row")`})
	text := Serialize(f.Root, lib, ObjectFrom("$s", 1.0))
	if text != `root = Stack([Text("a - (b - c)"), Col("x", [1, -2])], "row")`+"\n$s = 1" {
		t.Fatalf("serialize: %q", text)
	}
	s := &serializer{lib: lib}
	n, _ := parseExpression(mustTokens("a - (b - c) * -(d + e)"), nil, nil, nil)
	if got := s.node(n); got != "a - (b - c) * -(d + e)" {
		t.Fatalf("precedence: %q", got)
	}
}

func mustTokens(src string) []token {
	toks, _ := tokenize(src, false, 0)
	return toks
}

func TestQueriesAndMutations(t *testing.T) {
	in := newFixtureInstance(t)
	prog := `$s = "open"
root = Stack([Metric("n", @Count(q.rows)), Button("Go", Action([@Run(m), @ToAssistant("done")]))])
q = Query("list", {status: $s}, {rows: []})
m = Mutation("make", {s: $s})`
	in.Update(Update{Program: prog, Streaming: true})
	if len(in.PendingFetches()) != 0 {
		t.Fatal("query ran while streaming")
	}
	f := in.Update(Update{Program: prog})
	reqs := in.PendingFetches()
	if len(reqs) != 1 || reqs[0].Tool != "list" || string(EncodeJSON(reqs[0].Args)) != `{"status":"open"}` {
		t.Fatalf("fetch %+v", reqs)
	}
	in.ResolveFetch(reqs[0], ObjectFrom("rows", []Value{1.0, 2.0}), nil)
	f = in.Render()
	if f.Root.Children()[0].Prop("value") != 2.0 {
		t.Fatalf("query result not visible: %v", f.Root.Children()[0].Prop("value"))
	}
	_ = in.SetState("$s", "closed")
	reqs = in.PendingFetches()
	if len(reqs) != 1 || in.Render().Root.Children()[0].Prop("value") != 2.0 {
		t.Fatal("previous result must stay visible while refetching")
	}
	in.ResolveFetch(reqs[0], nil, &ToolNotFoundError{Tool: "list", Available: []string{"x"}})
	if codes(in.Render().Errors)["tool-not-found"] != 1 {
		t.Fatal("tool-not-found")
	}
	run := in.Dispatch("Go", "", f.Root.Children()[1].Prop("action"))
	eff, _ := run.Next()
	if eff == nil || eff.Kind != EffectRunMutation || string(EncodeJSON(eff.Mutation.Args)) != `{"s":"closed"}` {
		t.Fatalf("mutation %+v", eff)
	}
	run.ResolveMutation(ObjectFrom("id", 1.0), nil)
	eff, _ = run.Next()
	if eff == nil || eff.Event.HumanFriendlyMessage != "done" {
		t.Fatal("step after mutation")
	}
}

// FuzzParse checks that arbitrary input never panics and that streaming in
// two chunks equals a batch parse.
func FuzzParse(f *testing.F) {
	lib := loadFixtureLibrary(f)
	for _, id := range fixtureCases(f) {
		files := readCase(&testing.T{}, filepath.Join(fixturesDir, id))
		if s, ok := files["input.oui"]; ok {
			f.Add(s, uint16(len(s)/2))
		}
	}
	f.Add("root = Stack([Text(\"a\\", uint16(5))
	f.Add("```openui\nroot = $a ? @Each(x, \"i\", Text(i)) : null\n", uint16(9))
	f.Fuzz(func(t *testing.T, src string, cut uint16) {
		if len(src) > 4096 {
			return
		}
		in := NewInstance(lib, Options{Limits: Limits{MaxEvalSteps: 20000}})
		k := int(cut) % (len(src) + 1)
		in.Update(Update{Program: src[:k], Streaming: true})
		fs := in.Update(Update{Program: src})
		batch := NewInstance(lib, Options{Limits: Limits{MaxEvalSteps: 20000}})
		fb := batch.Update(Update{Program: src})
		if js(norm(summarize(in, fs.Root))) != js(norm(summarize(batch, fb.Root))) {
			t.Fatalf("stream/batch differ for %q at %d", src, k)
		}
		_, _ = in.PersistState()
		if fs.Root != nil {
			for _, el := range fs.Root.Children() {
				if p, ok := el.Prop("action").(*ActionPlan); ok {
					run := in.Dispatch("x", el.FormName, p)
					for range 8 {
						if e, err := run.Next(); e == nil || err != nil {
							break
						}
					}
				}
			}
		}
	})
}
