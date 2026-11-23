package lang

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

const fixturesDir = "testdata/spec/fixtures/1.0"

func loadFixtureLibrary(t testing.TB) *Library {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(fixturesDir, "library.json"))
	if err != nil {
		t.Fatal(err)
	}
	lib, err := LoadLibrarySpec(data, map[string]FuncImpl{
		"Upper": func(args *Object, _ *FuncEnv) (Value, error) {
			s := ""
			if v := args.Value("text"); v != nil {
				s = ToString(v)
			}
			return strings.ToUpper(s), nil
		},
	}, UpstreamProfile)
	if err != nil {
		t.Fatal(err)
	}
	return lib
}

// summarize renders the canonical tree summary of the fixture README.
func summarize(in *Instance, v Value) any {
	switch t := v.(type) {
	case nil:
		return nil
	case bool, string:
		return t
	case float64:
		if math.IsNaN(t) || math.IsInf(t, 0) {
			return nil // JSON.stringify writes null
		}
		return t
	case []Value:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = summarize(in, x)
		}
		return out
	case *Element:
		if t == nil {
			return nil
		}
		props := map[string]any{}
		for _, k := range t.Props.Keys() {
			if s := summarize(in, t.Props.Value(k)); s != nil {
				props[k] = s
			}
		}
		return map[string]any{"type": t.Type, "props": props}
	case *ActionPlan:
		acts := []any{}
		for _, st := range in.DescribePlan(t) {
			args := make([]any, len(st.Args))
			for i, a := range st.Args {
				args[i] = summarize(in, a)
			}
			acts = append(acts, map[string]any{"action": st.Action, "args": args})
		}
		return map[string]any{"actions": acts}
	case *Binding:
		return map[string]any{"$binding": t.Target}
	case *Object:
		out := map[string]any{}
		for _, k := range t.Keys() {
			out[k] = summarize(in, t.Value(k))
		}
		return out
	}
	return nil
}

// norm round-trips through JSON so expected and actual compare alike.
func norm(v any) any {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	var out any
	_ = json.Unmarshal(raw, &out)
	return out
}

func valueJSON(v Value) any {
	var out any
	_ = json.Unmarshal(EncodeJSON(v), &out)
	return out
}

type caseResult struct {
	root         any
	errors       []map[string]any
	errorObjects []map[string]any
	unresolved   []string
	orphans      []string
	incomplete   bool
	state        any
	tree         *Element
}

func evaluateCase(in *Instance, frame *Frame) caseResult {
	res := in.Result()
	r := caseResult{
		root:       norm(summarize(in, frame.Root)),
		unresolved: res.Unresolved,
		orphans:    res.Orphaned,
		incomplete: res.Incomplete,
		state:      valueJSON(res.StateDefaults),
		tree:       frame.Root,
	}
	if frame.Root == nil {
		r.root = nil
	}
	for _, d := range frame.Errors {
		e := map[string]any{"code": d.Code}
		if d.StatementID != "" {
			e["statementId"] = d.StatementID
		}
		r.errors = append(r.errors, e)
		obj := map[string]any{"source": d.Source, "code": d.Code}
		if d.StatementID != "" {
			obj["statementId"] = d.StatementID
		}
		if d.Component != "" {
			obj["component"] = d.Component
		}
		if d.Path != "" {
			obj["path"] = d.Path
		}
		r.errorObjects = append(r.errorObjects, obj)
	}
	return r
}

// claims matches each expected entry to a distinct actual entry agreeing on
// the expected keys (missing actual keys read as null).
func claims(expected []any, actual []map[string]any) bool {
	pool := append([]map[string]any(nil), actual...)
	exp := append([]any(nil), expected...)
	sort.SliceStable(exp, func(i, j int) bool { return len(exp[i].(map[string]any)) > len(exp[j].(map[string]any)) })
	for _, e := range exp {
		em := e.(map[string]any)
		found := -1
		for i, a := range pool {
			ok := true
			for k, v := range em {
				if !reflect.DeepEqual(norm(v), norm(a[k])) {
					ok = false
					break
				}
			}
			if ok {
				found = i
				break
			}
		}
		if found < 0 {
			return false
		}
		pool = append(pool[:found], pool[found+1:]...)
	}
	return true
}

func asSet(v any) []string {
	var out []string
	switch t := v.(type) {
	case []any:
		for _, x := range t {
			out = append(out, x.(string))
		}
	case []string:
		out = append(out, t...)
	}
	seen := map[string]bool{}
	var uniq []string
	for _, s := range out {
		if !seen[s] {
			seen[s] = true
			uniq = append(uniq, s)
		}
	}
	sort.Strings(uniq)
	return uniq
}

func checkCase(t *testing.T, lib *Library, id string, expected map[string]any, r caseResult, statements []string) []string {
	var diffs []string
	if !reflect.DeepEqual(norm(expected["root"]), r.root) {
		diffs = append(diffs, fmt.Sprintf("root: expected %s, got %s", js(expected["root"]), js(r.root)))
	}
	expErrs, _ := expected["errors"].([]any)
	if len(expErrs) != len(r.errors) || !claims(expErrs, r.errors) {
		diffs = append(diffs, fmt.Sprintf("errors: expected %s, got %s", js(expErrs), js(r.errors)))
	}
	if !reflect.DeepEqual(asSet(expected["unresolved"]), asSet(r.unresolved)) {
		diffs = append(diffs, fmt.Sprintf("unresolved: expected %s, got %s", js(expected["unresolved"]), js(r.unresolved)))
	}
	if !reflect.DeepEqual(asSet(expected["orphans"]), asSet(r.orphans)) {
		diffs = append(diffs, fmt.Sprintf("orphans: expected %s, got %s", js(expected["orphans"]), js(r.orphans)))
	}
	if expected["incomplete"] != r.incomplete {
		diffs = append(diffs, fmt.Sprintf("incomplete: expected %v, got %v", expected["incomplete"], r.incomplete))
	}
	if s, ok := expected["state"]; ok && !reflect.DeepEqual(norm(s), norm(r.state)) {
		diffs = append(diffs, fmt.Sprintf("state: expected %s, got %s", js(s), js(r.state)))
	}
	if s, ok := expected["statements"]; ok && !reflect.DeepEqual(norm(s), norm(statements)) {
		diffs = append(diffs, fmt.Sprintf("statements: expected %s, got %s", js(s), js(statements)))
	}
	if eo, ok := expected["errorObjects"].([]any); ok && !claims(eo, r.errorObjects) {
		diffs = append(diffs, fmt.Sprintf("errorObjects: expected %s, got %s", js(eo), js(r.errorObjects)))
	}
	if strings.Contains(id, "-roundtrip-") && r.tree != nil {
		text := Serialize(r.tree, lib, nil)
		in := NewInstance(lib, Options{})
		again := evaluateCase(in, in.Update(Update{Program: text}))
		if !reflect.DeepEqual(again.root, norm(expected["root"])) {
			diffs = append(diffs, fmt.Sprintf("roundtrip: %q parses to %s", text, js(again.root)))
		}
	}
	return diffs
}

func js(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}

var stmtLine = regexp.MustCompile(`^(\$?\w+)\s*=`)

func runProgramCase(t *testing.T, lib *Library, id string, files map[string]string) []string {
	var expected map[string]any
	if err := json.Unmarshal([]byte(files["expected.json"]), &expected); err != nil {
		t.Fatal(err)
	}
	if c, ok := files["chunks.json"]; ok {
		var chunks []string
		if err := json.Unmarshal([]byte(c), &chunks); err != nil {
			t.Fatal(err)
		}
		in := NewInstance(lib, Options{})
		text := ""
		var frame *Frame
		for i, ch := range chunks {
			text += ch
			frame = in.Update(Update{Program: text, Streaming: i < len(chunks)-1})
		}
		var diffs []string
		for _, d := range checkCase(t, lib, id, expected, evaluateCase(in, frame), nil) {
			diffs = append(diffs, "streamed "+d)
		}
		batch := NewInstance(lib, Options{})
		for _, d := range checkCase(t, lib, id, expected, evaluateCase(batch, batch.Update(Update{Program: strings.Join(chunks, "")})), nil) {
			diffs = append(diffs, "batch "+d)
		}
		return diffs
	}
	program := files["input.oui"]
	var statements []string
	if patch, ok := files["patch.oui"]; ok {
		program = MergeStatements(program, patch, lib)
		for _, line := range strings.Split(program, "\n") {
			if m := stmtLine.FindStringSubmatchIndex(line); m != nil {
				if end := m[1]; end < len(line) && line[end] == '=' {
					continue
				}
				statements = append(statements, line[m[2]:m[3]])
			}
		}
	}
	in := NewInstance(lib, Options{})
	return checkCase(t, lib, id, expected, evaluateCase(in, in.Update(Update{Program: program})), statements)
}

func runValidationCase(t *testing.T, files map[string]string) []string {
	v, err := ParseJSON([]byte(files["validation.json"]))
	if err != nil {
		t.Fatal(err)
	}
	var diffs []string
	for i, c := range v.(*Object).Value("cases").([]Value) {
		co := c.(*Object)
		rules, _ := co.Value("rules").(*Object)
		msg := ValidateRules(co.Value("value"), rules)
		if (msg == "") != co.Value("valid").(bool) {
			diffs = append(diffs, fmt.Sprintf("case %d %s %s: got %q", i, EncodeJSON(rules), EncodeJSON(co.Value("value")), msg))
		}
	}
	return diffs
}

// ── interactions.json (fixture README, "Action and form cases") ──

func findElement(root *Element, id string) *Element {
	if root == nil {
		return nil
	}
	if root.StatementID == id {
		return root
	}
	for _, c := range root.Children() {
		if el := findElement(c, id); el != nil {
			return el
		}
	}
	return nil
}

type fixtureHost struct {
	in        *Instance
	tools     map[string]Value
	toolCalls []map[string]any
	events    []HostEvent
}

func (h *fixtureHost) callTool(name string, args Value) (Value, error) {
	res, ok := h.tools[name]
	if !ok {
		var avail []string
		for k := range h.tools {
			avail = append(avail, k)
		}
		sort.Strings(avail)
		return nil, &ToolNotFoundError{Tool: name, Available: avail}
	}
	h.toolCalls = append(h.toolCalls, map[string]any{"tool": name, "args": valueJSON(args)})
	return res, nil
}

func (h *fixtureHost) settle() {
	for range 5 {
		reqs := h.in.PendingFetches()
		if len(reqs) == 0 {
			return
		}
		for _, r := range reqs {
			data, err := h.callTool(r.Tool, r.Args)
			h.in.ResolveFetch(r, data, err)
		}
	}
}

func (h *fixtureHost) run(run *ActionRun) {
	for {
		eff, err := run.Next()
		if err != nil || eff == nil {
			return
		}
		switch eff.Kind {
		case EffectRunMutation:
			data, err := h.callTool(eff.Mutation.Tool, eff.Mutation.Args)
			run.ResolveMutation(data, err)
		case EffectRefetch:
		default:
			h.events = append(h.events, eff.Event)
		}
	}
}

func eventJSON(e HostEvent) map[string]any {
	m := map[string]any{"type": e.Type, "humanFriendlyMessage": e.HumanFriendlyMessage, "params": valueJSON(e.Params)}
	if e.FormName != "" {
		m["formName"] = e.FormName
	}
	if e.FormState != nil {
		m["formState"] = valueJSON(e.FormState)
	}
	return m
}

func runInteractions(t *testing.T, lib *Library, program string, spec map[string]any) []string {
	h := &fixtureHost{tools: map[string]Value{}}
	steps, _ := spec["interactions"].([]any)
	expected, _ := spec["expected"].(map[string]any)
	i := 0
	for ; i < len(steps); i++ {
		s := steps[i].(map[string]any)
		if _, ok := s["tool"]; !ok {
			break
		}
		h.tools[s["tool"].(string)] = FromGo(s["result"])
	}
	h.in = NewInstance(lib, Options{})
	frame := h.in.Update(Update{Program: program})
	h.settle()
	var diffs []string
	for ; i < len(steps); i++ {
		s := steps[i].(map[string]any)
		if tool, ok := s["tool"]; ok {
			h.tools[tool.(string)] = FromGo(s["result"])
			continue
		}
		frame = h.in.Render()
		if ty, ok := s["type"].([]any); ok {
			el := findElement(frame.Root, ty[0].(string))
			if el == nil {
				diffs = append(diffs, fmt.Sprintf("step %d: nothing rendered for %v", i, ty[0]))
				break
			}
			text := ty[1].(string)
			if b, ok := el.Prop("value").(*Binding); ok {
				if err := h.in.SetBinding(b, text); err != nil {
					diffs = append(diffs, err.Error())
				}
			} else {
				name, _ := el.Prop("name").(string)
				_ = h.in.SetField(el.FormName, "input", name, text)
			}
		} else {
			id := s["click"].(string)
			el := findElement(frame.Root, id)
			if el == nil {
				diffs = append(diffs, fmt.Sprintf("step %d: nothing rendered for %s", i, id))
				break
			}
			label, _ := el.Prop("label").(string)
			h.run(h.in.Dispatch(label, el.FormName, el.Prop("action")))
		}
		h.settle()
	}
	want, _ := expected["events"].([]any)
	ok := len(want) == len(h.events)
	for k := 0; ok && k < len(want); k++ {
		got := eventJSON(h.events[k])
		for key, v := range want[k].(map[string]any) {
			if !reflect.DeepEqual(norm(v), norm(got[key])) {
				ok = false
			}
		}
	}
	if !ok {
		var got []any
		for _, e := range h.events {
			got = append(got, eventJSON(e))
		}
		diffs = append(diffs, fmt.Sprintf("events: expected %s, got %s", js(want), js(got)))
	}
	if st, ok := expected["state"].(map[string]any); ok {
		for k, v := range st {
			got := valueJSON(unwrapField(h.in.Store().Get(k)))
			if !reflect.DeepEqual(norm(v), got) {
				diffs = append(diffs, fmt.Sprintf("state %s: expected %s, got %s", k, js(v), js(got)))
			}
		}
	}
	if tc, ok := expected["toolCalls"]; ok {
		var calls []map[string]any
		for k, c := range h.toolCalls {
			if k > 0 && reflect.DeepEqual(norm(c), norm(h.toolCalls[k-1])) {
				continue
			}
			calls = append(calls, c)
		}
		if calls == nil {
			calls = []map[string]any{}
		}
		if !reflect.DeepEqual(norm(tc), norm(calls)) {
			diffs = append(diffs, fmt.Sprintf("toolCalls: expected %s, got %s", js(tc), js(calls)))
		}
	}
	return diffs
}

func readCase(t *testing.T, dir string) map[string]string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, e := range entries {
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		files[e.Name()] = strings.ReplaceAll(string(raw), "\r\n", "\n")
	}
	return files
}

func fixtureCases(t testing.TB) []string {
	areas, err := os.ReadDir(fixturesDir)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, a := range areas {
		if !a.IsDir() {
			continue
		}
		cases, _ := os.ReadDir(filepath.Join(fixturesDir, a.Name()))
		for _, c := range cases {
			if c.IsDir() {
				ids = append(ids, a.Name()+"/"+c.Name())
			}
		}
	}
	sort.Strings(ids)
	return ids
}

// TestConformance runs the 59 pinned OpenUI 1.0 conformance cases (program
// level) and the interactions.json cases.
func TestConformance(t *testing.T) {
	lib := loadFixtureLibrary(t)
	ids := fixtureCases(t)
	pass, total, ipass, itotal := 0, 0, 0, 0
	for _, id := range ids {
		files := readCase(t, filepath.Join(fixturesDir, id))
		total++
		var diffs []string
		if _, ok := files["validation.json"]; ok {
			diffs = runValidationCase(t, files)
		} else {
			diffs = runProgramCase(t, lib, id, files)
		}
		if len(diffs) == 0 {
			pass++
		} else {
			t.Errorf("%s\n  %s", id, strings.Join(diffs, "\n  "))
		}
		if raw, ok := files["interactions.json"]; ok {
			itotal++
			var spec map[string]any
			if err := json.Unmarshal([]byte(raw), &spec); err != nil {
				t.Fatal(err)
			}
			if d := runInteractions(t, lib, files["input.oui"], spec); len(d) == 0 {
				ipass++
			} else {
				t.Errorf("%s#interactions\n  %s", id, strings.Join(d, "\n  "))
			}
		}
	}
	t.Logf("conformance: %d/%d program cases, %d/%d interaction cases", pass, total, ipass, itotal)
	if total != 59 {
		t.Errorf("expected 59 fixture cases, found %d", total)
	}
}

var _ = errors.New

// TestConformanceHarnessDetectsMismatch guards the harness itself: a wrong
// expectation must produce diffs.
func TestConformanceHarnessDetectsMismatch(t *testing.T) {
	lib := loadFixtureLibrary(t)
	files := readCase(t, filepath.Join(fixturesDir, "entry/001-root-statement"))
	files["expected.json"] = strings.Replace(files["expected.json"], `"Hi"`, `"Bye"`, 1)
	if d := runProgramCase(t, lib, "entry/001-root-statement", files); len(d) == 0 {
		t.Fatal("a changed expected root was not detected")
	}
	files = readCase(t, filepath.Join(fixturesDir, "errors/001-unknown-component-dropped"))
	files["expected.json"] = strings.Replace(files["expected.json"], `"unknown-component"`, `"type-mismatch"`, 1)
	if d := runProgramCase(t, lib, "errors/001", files); len(d) == 0 {
		t.Fatal("a changed expected error was not detected")
	}
}
