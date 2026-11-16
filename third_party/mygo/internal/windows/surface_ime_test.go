//go:build windows && (amd64 || arm64)

package windows

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

type stubSurfaceTextClient struct {
	text   string
	marked *platform.TextRange
	sel    platform.TextSelection
}

func (c *stubSurfaceTextClient) TextForRange(r platform.TextRange) (string, platform.TextRange) {
	r = platform.NormalizeTextRange(r)
	n := platform.UTF16Len(c.text)
	r.Start, r.End = min(r.Start, n), min(r.End, n)
	a, z := platform.UTF16ByteOffset(c.text, r.Start), platform.UTF16ByteOffset(c.text, r.End)
	return c.text[a:z], r
}

func (c *stubSurfaceTextClient) Selection() platform.TextSelection { return c.sel }

func (c *stubSurfaceTextClient) MarkedRange() (platform.TextRange, bool) {
	if c.marked == nil {
		return platform.TextRange{}, false
	}
	return *c.marked, true
}

func (c *stubSurfaceTextClient) ReplaceText(r *platform.TextRange, text string) {
	wanted := c.sel.Range
	if r != nil {
		wanted = *r
	} else if c.marked != nil {
		wanted = *c.marked
	}
	wanted = platform.NormalizeTextRange(wanted)
	a, z := platform.UTF16ByteOffset(c.text, wanted.Start), platform.UTF16ByteOffset(c.text, wanted.End)
	c.text = c.text[:a] + text + c.text[z:]
	end := wanted.Start + platform.UTF16Len(text)
	c.sel = platform.TextSelection{Range: platform.TextRange{Start: end, End: end}}
	c.marked = nil
}

func (c *stubSurfaceTextClient) SetMarkedText(r *platform.TextRange, text string, selected platform.TextRange) {
	wanted := c.sel.Range
	if r != nil {
		wanted = *r
	} else if c.marked != nil {
		wanted = *c.marked
	}
	wanted = platform.NormalizeTextRange(wanted)
	a, z := platform.UTF16ByteOffset(c.text, wanted.Start), platform.UTF16ByteOffset(c.text, wanted.End)
	c.text = c.text[:a] + text + c.text[z:]
	mark := platform.TextRange{Start: wanted.Start, End: wanted.Start + platform.UTF16Len(text)}
	c.marked = &mark
	c.sel = platform.TextSelection{Range: platform.TextRange{Start: mark.Start + selected.Start, End: mark.Start + selected.End}}
}

func (c *stubSurfaceTextClient) UnmarkText() { c.marked = nil }

func (c *stubSurfaceTextClient) BoundsForRange(platform.TextRange) (platform.RectF, platform.TextRange, bool) {
	return platform.RectF{}, platform.TextRange{}, false
}

func (c *stubSurfaceTextClient) IndexForPoint(float64, float64) (int, bool) { return 0, false }

func TestSurfaceKeyDownPreservesPendingIMECompositionAcrossProcessAndUntakenKeys(t *testing.T) {
	client := &stubSurfaceTextClient{text: "prefix ", sel: platform.TextSelection{Range: platform.TextRange{Start: 7, End: 7}}}
	h := &surfaceKeyHandler{taken: false}
	s := &surface{
		w:     &window{opts: &platform.WindowOptions{}, h: h},
		input: platform.TextInputState{Active: true, Client: client},
	}

	s.composed(platform.SurfaceEvent{Kind: platform.TextComposition, Text: "zhong", Caret: 5})
	if _, marked := client.MarkedRange(); !marked || client.text != "prefix zhong" {
		t.Fatalf("unexpected preedit state: text=%q marked=%v", client.text, client.marked)
	}

	// Unmark first, as IMM32 may dispatch WM_IME_ENDCOMPOSITION before GCS_RESULTSTR.
	s.message(0, wmImeEndComp, 0, 0)
	if _, marked := client.MarkedRange(); marked {
		t.Fatal("wmImeEndComp did not unmark text")
	}

	// Intervening VK_PROCESSKEY (0xE5), modifier key, untaken Enter, and '\r' WM_CHAR
	// must NOT destroy clientComposition.pending.
	s.message(0, wmKeyDown, 0xE5, 0x001C0001)
	s.message(0, wmKeyDown, vkShift, 1)
	s.message(0, wmKeyDown, 0x0D, 0x001C0001)
	s.message(0, wmChar, '\r', 1)

	s.composed(platform.SurfaceEvent{Kind: platform.TextInput, Text: "中"})
	if client.text != "prefix 中" {
		t.Fatalf("pending composition range lost before commit: text=%q want %q", client.text, "prefix 中")
	}
}

func TestSurfacePendingIMECompositionResetsOnConsumedKeyCharAndClientSwitch(t *testing.T) {
	clientA := &stubSurfaceTextClient{text: "ab", sel: platform.TextSelection{Range: platform.TextRange{Start: 2, End: 2}}}
	h := &surfaceKeyHandler{taken: true}
	s := &surface{
		w:     &window{opts: &platform.WindowOptions{}, h: h},
		input: platform.TextInputState{Active: true, Client: clientA},
	}

	// Case 1: consumed non-IME keydown clears stale pending range.
	s.composed(platform.SurfaceEvent{Kind: platform.TextComposition, Text: "xy", Caret: 2})
	s.message(0, wmImeEndComp, 0, 0)
	s.message(0, wmKeyDown, 0x08, 1)
	s.composed(platform.SurfaceEvent{Kind: platform.TextInput, Text: "新"})
	if clientA.text != "abxy新" {
		t.Fatalf("consumed Backspace did not clear stale pending range: %q", clientA.text)
	}

	// Case 2: direct printable WM_CHAR clears stale pending range.
	h.taken = false
	s.composed(platform.SurfaceEvent{Kind: platform.TextComposition, Text: "zz", Caret: 2})
	s.message(0, wmImeEndComp, 0, 0)
	s.message(0, wmChar, '!', 1)
	s.composed(platform.SurfaceEvent{Kind: platform.TextInput, Text: "字"})
	if clientA.text != "abxy新zz字" {
		t.Fatalf("printable WM_CHAR did not clear stale pending range: %q", clientA.text)
	}

	// Case 3: switching active TextInputClient resets pending composition cleanly.
	clientB := &stubSurfaceTextClient{}
	s.composed(platform.SurfaceEvent{Kind: platform.TextComposition, Text: "pp", Caret: 2})
	s.message(0, wmImeEndComp, 0, 0)
	s.SetTextInput(platform.TextInputState{Active: true, Client: clientB})
	s.composed(platform.SurfaceEvent{Kind: platform.TextInput, Text: "词"})
	if clientB.text != "词" {
		t.Fatalf("switching client leaked old pending range: %q", clientB.text)
	}
}
