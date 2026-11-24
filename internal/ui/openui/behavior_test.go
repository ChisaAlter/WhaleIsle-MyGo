package openui

import (
	"strings"
	"sync"
	"testing"
	"time"

	"whaleisle/internal/errcopy"
	"whaleisle/internal/openui/card"
	"whaleisle/internal/openui/lang"
)

func TestBindingIsTwoWay(t *testing.T) {
	h := newHarness(t, "root = Col([Input($name, \"Name\", \"text\", \"Your name\"), Text(\"Hello \" + $name), Button(\"Reset\", @Set($name, \"Zed\"))])\n$name = \"\"", card.Config{})
	h.mustText("Hello ")
	h.click("Your name")
	h.tt.Type("Bob")
	h.frame()
	h.frame()
	h.mustText("Hello Bob")
	if v := h.card.Frame().Root.Children()[0].Prop("value").(*lang.Binding).Value; v != "Bob" {
		t.Fatalf("$name = %v", v)
	}
	// A state write elsewhere replaces the input's text.
	h.click("Reset")
	h.mustText("Hello Zed")
	h.mustText("Zed")
}

func TestNumberInputStoresNumbers(t *testing.T) {
	h := newHarness(t, "root = Col([Input($n, \"Count\", \"number\", \"0\"), Text(\"Double \" + $n * 2)])\n$n = null", card.Config{})
	h.click("0")
	h.tt.Type("21")
	h.frame()
	h.frame()
	h.mustText("Double 42")
}

func TestCheckboxAndSelectWriteState(t *testing.T) {
	h := newHarness(t, "root = Col([Checkbox($ok, \"Agree\"), Text($ok ? \"yes\" : \"no\"), SegmentedControl($s, [\"Day\", \"Week\"]), Text(\"mode \" + $s)])\n$ok = false\n$s = \"Day\"", card.Config{})
	h.mustText("no")
	h.click("Agree")
	h.mustText("yes")
	h.click("Week")
	h.mustText("mode Week")
}

func TestToAssistantSubmitsOncePerClick(t *testing.T) {
	h := newHarness(t, "root = Col([Button(\"Send\", @ToAssistant(\"Hi \" + $name, {id: 7}))])\n$name = \"Ann\"", card.Config{})
	h.mustText("Send")
	for i := 0; i < 5; i++ {
		h.frame()
	}
	if len(h.host.submitted) != 0 {
		t.Fatal("submitted without a click")
	}
	h.click("Send")
	h.host.wait(t)
	for i := 0; i < 5; i++ {
		h.frame()
	}
	h.host.mu.Lock()
	got := append([]string(nil), h.host.submitted...)
	h.host.mu.Unlock()
	if len(got) != 1 || !strings.HasPrefix(got[0], "Hi Ann") || !strings.Contains(got[0], `OpenUI context:`+"\n"+`{"id":7}`) {
		t.Fatalf("submitted = %q", got)
	}
	h.click("Send")
	h.host.wait(t)
	if len(h.host.submitted) != 2 {
		t.Fatalf("second click submitted %d times total", len(h.host.submitted))
	}
}

func TestDefaultButtonSendsLabelWithFormState(t *testing.T) {
	h := newHarness(t, "root = Form([Input($email, \"Email\", \"text\", \"you@\"), Button(\"Subscribe\")])\n$email = \"a@b.co\"", card.Config{})
	h.click("Subscribe")
	h.host.wait(t)
	got := h.host.submitted[0]
	if !strings.HasPrefix(got, "Subscribe\n\nOpenUI form state:\n") || !strings.Contains(got, `"$email":"a@b.co"`) {
		t.Fatalf("submitted = %q", got)
	}
}

func TestSubmitFailureShowsASentence(t *testing.T) {
	h := newHarness(t, `root = Col([Button("Send", @ToAssistant("x"))])`, card.Config{})
	h.host.submitErr = errBoom
	h.click("Send")
	h.host.wait(t)
	deadline := time.Now().Add(5 * time.Second)
	for h.card.Failure(h.card.Frame().Root.Children()[0].Identity()) == nil && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	h.mustText(errcopy.Sentence(errBoom))
	h.noText("connection refused")
}

func TestOpenURLValidatedAndCopy(t *testing.T) {
	h := newHarness(t, `root = Col([Button("Open", @OpenUrl("https://example.com/a")), Button("Bad", @OpenUrl("ftp://example.com")), Button("Copy", @CopyToClipboard("copied text"))])`, card.Config{})
	h.click("Open")
	h.host.wait(t)
	h.click("Copy")
	h.host.wait(t)
	h.click("Bad")
	h.frame()
	if len(h.host.opened) != 1 || h.host.opened[0] != "https://example.com/a" {
		t.Fatalf("opened = %q", h.host.opened)
	}
	if len(h.host.copied) != 1 || h.host.copied[0] != "copied text" {
		t.Fatalf("copied = %q", h.host.copied)
	}
	bad := h.card.Frame().Root.Children()[1]
	if h.card.Failure(bad.Identity()) == nil {
		t.Fatal("invalid URL did not fail")
	}
	h.noText("links require")
}

func TestLocalActionsApplyAtOnce(t *testing.T) {
	h := newHarness(t, "root = Col([Text(\"n=\" + $n), Button(\"Inc\", @Set($n, $n + 1))])\n$n = 0", card.Config{})
	h.click("Inc")
	h.mustText("n=1")
	h.click("Inc")
	h.mustText("n=2")
	if len(h.host.submitted)+len(h.host.opened) != 0 {
		t.Fatal("a local action reached the host")
	}
}

func TestStatePersistsAcrossCards(t *testing.T) {
	dir := t.TempDir()
	id := strings.Repeat("ab", 32)
	prog := "root = Col([Checkbox($ok, \"Agree\"), Text($ok ? \"on\" : \"off\"), Flashcard(\"Q\", \"A\")])\n$ok = false"
	h := newHarness(t, prog, card.Config{StateDir: dir, StateID: id})
	h.mustText("off")
	h.click("Agree")
	h.click("Q")
	h.mustText("on")
	h.mustText("A")
	h.card.Flush()

	h2 := newHarness(t, prog, card.Config{StateDir: dir, StateID: id})
	deadline := time.Now().Add(5 * time.Second)
	for !h2.card.Loaded() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	h2.mustText("on")
	h2.mustText("A")
}

func TestStreamingKeepsTreeAndDisablesActions(t *testing.T) {
	now := time.Unix(1000, 0)
	var clockMu sync.Mutex
	clock := func() time.Time {
		clockMu.Lock()
		defer clockMu.Unlock()
		return now
	}
	h := newHarness(t, "", card.Config{Now: clock, Coalesce: time.Millisecond})
	steps := []string{
		`root = Col([Title("Plan"), Te`,
		`root = Col([Title("Plan"), Text("Step one"), Button("Go", @ToAssistant("go"))`,
		`root = Col([Title("Plan"), Text("Step one"), Button("Go", @ToAssistant("go"))])`,
	}
	var last uint64
	for _, s := range steps {
		clockMu.Lock()
		now = now.Add(time.Second)
		clockMu.Unlock()
		h.card.SetSource("whale-ui", s, true)
		h.frame()
		f := h.card.Frame()
		if f.Revision < last {
			t.Fatalf("revision went back: %d < %d", f.Revision, last)
		}
		last = f.Revision
		h.mustText("Plan")
	}
	h.mustText("Go")
	h.click("Go")
	if len(h.host.submitted) != 0 {
		t.Fatal("a streaming card ran an action")
	}
	h.card.SetSource("whale-ui", steps[2], false)
	h.click("Go")
	h.host.wait(t)
}

func TestUnchangedCardDoesNoEvaluation(t *testing.T) {
	h := newHarness(t, "root = Col([Title(\"Static\"), Text(\"n=\" + $n), Table([\"a\"], [[\"b\"]])])\n$n = 1", card.Config{})
	h.mustText("Static")
	before := h.card.Evaluations()
	for i := 0; i < 20; i++ {
		h.card.SetSource("whale-ui", "root = Col([Title(\"Static\"), Text(\"n=\" + $n), Table([\"a\"], [[\"b\"]])])\n$n = 1", false)
		h.frame()
	}
	if got := h.card.Evaluations(); got != before {
		t.Fatalf("evaluations %d → %d on unchanged frames", before, got)
	}
}
