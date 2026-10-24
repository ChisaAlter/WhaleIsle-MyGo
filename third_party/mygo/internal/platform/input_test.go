package platform

import "testing"

type beforeInputStub struct{ fn func(KeyInput) bool }

func (s *beforeInputStub) SetBeforeInput(fn func(KeyInput) bool) { s.fn = fn }

// TestSetBeforeInputCapability covers the additive seam: a backend that
// implements the key-input interface accepts the callback, one that does not
// reports false so callers can tell the platform cannot report key input.
func TestSetBeforeInputCapability(t *testing.T) {
	got := KeyInput{}
	stub := &beforeInputStub{}
	if !SetBeforeInput(stub, func(in KeyInput) bool { got = in; return true }) {
		t.Fatal("a window with SetBeforeInput reported no support")
	}
	if stub.fn == nil {
		t.Fatal("the callback was not installed")
	}
	if !stub.fn(KeyInput{Key: "i", Control: true, Shift: true, Down: true}) {
		t.Fatal("the stub callback result was not returned")
	}
	if got.Key != "i" || !got.Control || !got.Shift || !got.Down {
		t.Fatalf("callback saw %+v", got)
	}
	if SetBeforeInput(struct{}{}, func(KeyInput) bool { return false }) {
		t.Fatal("a window without the interface reported support")
	}
}
