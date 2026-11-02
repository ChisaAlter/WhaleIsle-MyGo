package platform

import "testing"

func TestPrototypeMouseOptionalBackendAndCancelMasks(t *testing.T) {
	if BrowserMouseInputProvider(struct{}{}) != nil {
		t.Fatal("unsupported backend reported capability")
	}
	events := BrowserMouseCancelEvents(1 | 2 | 4)
	if len(events) != 5 {
		t.Fatalf("cancel=%+v", events)
	}
	want := []uint8{7, 6, 4, 0, 0}
	for i, e := range events {
		if err := ValidateBrowserMouseEvent(e); err != nil || e.Buttons != want[i] || e.X != -1 || e.Y != -1 {
			t.Fatalf("event%d=%+v error=%v", i, e, err)
		}
	}
	if events[1].Button != BrowserMouseLeft || events[2].Button != BrowserMouseRight || events[3].Button != BrowserMouseMiddle {
		t.Fatal("CDP mask order swapped")
	}
}
