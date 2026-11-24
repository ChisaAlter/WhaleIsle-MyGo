package openui

import (
	"strings"
	"testing"

	"whaleisle/internal/errcopy"
	"whaleisle/internal/openui/card"
	"whaleisle/internal/openui/lang/whaleui"
)

// gallery is a representative program per component family; every
// implemented component appears at least once.
var gallery = map[string]struct {
	program string
	texts   []string
}{
	"layout": {`root = Col([Title("Layout title"), Row([Text("Left"), Spacer(), Text("Right")], 2, "center", "between", true), Box([Text("In a box")], "row", 2, 3, "info", "md", true), Grid([Card([Text("Cell one")]), Card([Text("Cell two")], "soft")], 2, 200, 3), Carousel([Card([Text("Slide A")]), Card([Text("Slide B")])], 200), Divider(), Details("More details", [Text("Hidden body")], true)])`,
		[]string{"Layout title", "Left", "Right", "In a box", "Cell one", "Cell two", "Slide A", "More details", "Hidden body"}},
	"text": {`root = Col([Title("Big", "xl"), Text("Some **bold** text"), Caption("A caption"), Label("A label"), Badge("New", "success"), Link("Docs", "https://example.com/docs"), Markdown("# Heading\n\n- item one\n- item two"), Quote("To be", "Hamlet"), Code("fmt.Println(1)", "go"), List(["alpha", "beta"], "number"), List([ListItem("Step one", "first"), ListItem("Step two")], "bullet", true)])`,
		[]string{"Big", "A caption", "A label", "New", "Docs", "Heading", "item one", "To be", "— Hamlet", "1.", "alpha", "Step one", "first", "Step two"}},
	"inputs": {`root = Col([Input($name, "Name"), TextArea($bio, "Bio"), Select($c, ["Red", "Blue"], "Color"), SegmentedControl($seg, ["Day", "Week"]), Slider($v, 0, 10, 1, "Volume", "dB"), Checkbox($ok, "Agree"), RadioGroup($r, ["One", {value: "two", label: "Two", description: "second"}], "Pick"), RadioGroup($p, ["P1", "P2"], "Pills", "pills"), RadioGroup($q, [{value: "a", label: "Card A", description: "desc A"}], "Cards", "cards"), DatePicker($d, "When"), Timer($s, "Brew"), Form([Input($e, "Email")]), Button("Go"), IconButton("plus", null, "Add"), Pressable([Text("Press me")]), Popover("Open pop", [Text("Pop body")])])
$name = "Ann"
$bio = ""
$c = "Blue"
$seg = "Week"
$v = 3
$ok = true
$r = "two"
$p = "P1"
$q = "a"
$d = "2026-01-02"
$s = 90
$e = ""`,
		[]string{"Name", "Bio", "Color", "Blue", "Day", "Week", "Volume", "3dB", "Agree", "Pick", "One", "Two", "second", "P1", "Card A", "desc A", "When", "Brew", "01:30", "Email", "Go", "Press me", "Open pop"}},
	"data": {`root = Col([Table(["Item", "Qty"], [["Apples", 3], ["Pears", "12"]]), Progress(40, 100, "Done"), Icon("heart", "lg", "danger"), Icon("no-such-icon")])`,
		[]string{"Item", "Qty", "Apples", "3", "Pears", "12", "Done", "40%"}},
	"specialized": {`root = Col([Tabs({tabs: [{label: "First", items: [{type: "text", content: "Tab one body"}]}, {label: "Second", items: [{type: "text", content: "Tab two"}]}]}), Accordion({items: [{title: "Section A", items: [{type: "text", content: "A body"}]}]}), Steps({steps: [{title: "Plan", desc: "think"}, {title: "Build"}], current: 1}), Timeline({items: [{title: "Kickoff", time: "9:00"}]}), KeyValue({pairs: [{key: "Owner", value: "Ann"}]}), Json({value: {a: 1, b: [true]}}), FileTree({items: [{name: "src", type: "dir", children: [{name: "main.go"}]}]}), Quiz({question: "2+2?", options: ["3", {label: "4", correct: true}]}), Blocks({rows: [{label: "Row", cells: ["x", "y"]}]}), Flashcard("Front side", "Back side", "think")])`,
		[]string{"First", "Second", "Tab one body", "Section A", "Plan", "think", "Build", "Kickoff", "9:00", "Owner", "Ann", "src", "main.go", "2+2?", "4", "Row", "x", "Front side"}},
}

func TestGalleryRendersEveryFamily(t *testing.T) {
	for _, dark := range []bool{false, true} {
		for name, g := range gallery {
			t.Run(name, func(t *testing.T) {
				h := newHarness(t, g.program, card.Config{})
				h.dark = dark
				h.frame()
				if f := h.card.Frame(); f.Root == nil {
					t.Fatalf("no root: %+v", f.Errors)
				}
				for _, s := range g.texts {
					h.mustText(s)
				}
				// Nothing fell back to the placeholder.
				h.noText(OpenCompatLabel())
			})
		}
	}
}

func TestEveryImplementedComponentIsRegistered(t *testing.T) {
	want := strings.Fields(`Col Row Box Grid Card Carousel Divider Spacer Details Title Text Caption Label Badge Link Markdown Quote Code List ListItem
		Input TextArea Select SegmentedControl Slider Checkbox RadioGroup DatePicker Timer Form Button IconButton Pressable Popover
		Table Progress Icon Image SearchImage Flashcard Celebration Tabs Accordion Steps Timeline KeyValue Json FileTree Quiz Blocks Genui`)
	for _, n := range want {
		if _, ok := Lookup(n); !ok {
			t.Errorf("%s has no native view", n)
		}
	}
	lib, err := whaleui.Library()
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range Native() {
		if lib.Components[n] == nil {
			t.Errorf("registered %s is not a library component", n)
		}
	}
}

func TestUnknownAndWebOnlyComponentsShowThePlaceholder(t *testing.T) {
	h := newHarness(t, `root = Col([HtmlView("<b>x</b>", 200, "Widget"), YouTube("dQw4w9WgXcQ"), Mermaid({code: "graph TD; A-->B"})])`, card.Config{})
	h.mustText("Widget")
	h.mustText(OpenCompatLabel())
	if n := len(h.tt.FindAll(OpenCompatLabel())); n != 3 {
		t.Fatalf("placeholders = %d, want 3", n)
	}
	h.click(OpenCompatLabel())
	if h.compat != 1 {
		t.Fatalf("compat opens = %d", h.compat)
	}
}

func TestErrorFrameShowsAClassifiedSentence(t *testing.T) {
	h := newHarness(t, `root = 42`, card.Config{})
	h.frame()
	f := h.card.Frame()
	if f.Root != nil || len(f.Errors) == 0 {
		t.Fatalf("frame = %+v", f)
	}
	h.mustText(errcopy.Sentence(diagnosticError(f.Errors)))
	if !strings.Contains(errcopy.Sentence(diagnosticError(f.Errors)), "界面") {
		t.Fatal("diagnostics did not classify as an interface failure")
	}
	for _, d := range f.Errors {
		h.noText(d.Message)
	}
}

func TestLinkOpensOnlyOnClick(t *testing.T) {
	h := newHarness(t, `root = Col([Link("Docs", "https://example.com/docs"), Link("Bad", "javascript:alert(1)")])`, card.Config{})
	h.mustText("Bad")
	if len(h.links) != 0 {
		t.Fatal("link opened without a click")
	}
	h.click("Docs")
	h.click("Bad")
	if len(h.links) != 1 || h.links[0] != "https://example.com/docs" {
		t.Fatalf("links = %q", h.links)
	}
}
