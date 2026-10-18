//go:build linux && (amd64 || arm64)

package linux

import "testing"

func TestBlankEncoderEphemeralContext(t *testing.T) {
	oldNew, oldCheck, oldObject, oldType, oldUnref := blankContextNew, blankContextIsEphemeral, browserObjectNew, browserWebViewType, gObjectUnref
	t.Cleanup(func() {
		blankContextNew, blankContextIsEphemeral, browserObjectNew, browserWebViewType, gObjectUnref = oldNew, oldCheck, oldObject, oldType, oldUnref
	})
	blankContextNew = nil
	if _, err := newBlankPreviewContext(); err == nil {
		t.Fatal("missing native API accepted")
	}
	blankContextNew = func() ptr { return 12 }
	blankContextIsEphemeral = func(ptr) bool { return false }
	browserObjectNew = func(_ ptr, _ *byte, _ ptr, _ *byte, _ ptr, _ ptr) ptr { return 1 }
	browserWebViewType = func() ptr { return 1 }
	released := ptr(0)
	gObjectUnref = func(p ptr) { released = p }
	if _, err := newBlankPreviewContext(); err == nil || released != 12 {
		t.Fatal("persistent context accepted or leaked")
	}
	blankContextIsEphemeral = func(ptr) bool { return true }
	if got, err := newBlankPreviewContext(); err != nil || got != 12 {
		t.Fatal(got, err)
	}
	blankContextNew = func() ptr { return 0 }
	if _, err := newBlankPreviewContext(); err == nil {
		t.Fatal("null context accepted")
	}
}
