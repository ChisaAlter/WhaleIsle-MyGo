package ui

import "testing"

func TestTesterNativeInputDoesNotSettleAndRetainsAdapterGuards(t *testing.T) {
	buffer := NewTextBuffer("base")
	show, frames := true, 0
	tester := coreNewTester(func(c *context) {
		frames++
		coreColumn(c).Fill().Children(func() {
			if show {
				coreTextAreaBuffer(c, buffer).Fill().AutoFocus()
			} else {
				coreButton(c, "replacement").AutoFocus()
			}
		})
	}, 300, 140)
	defer tester.rt.close()
	client := tester.NativeTextInputClient()
	if client == nil {
		t.Fatal("focused native adapter missing")
	}
	before := frames
	r := TextInputRange{End: 4}
	client.ReplaceText(&r, "pending")
	if buffer.String() != "pending" || frames != before {
		t.Fatal("native test access bypassed immediate publish or settled a frame")
	}
	show = false
	tester.Frame()
	client.ReplaceText(nil, "stale mutation")
	if buffer.String() != "pending" || tester.NativeTextInputClient() != nil {
		t.Fatal("stale native adapter bypassed disposal/focus guards")
	}
}
