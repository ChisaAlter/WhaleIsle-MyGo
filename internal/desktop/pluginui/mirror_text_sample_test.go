package pluginui

import (
	"testing"
	"time"
)

func TestMirrorOlderCaptureCannotRollBackAcknowledgedOriginalText(t *testing.T) {
	_, client, focus := textFixture()
	old := time.Now().Add(-time.Second)
	client.applyTextSampleLocked(focus, old)
	acknowledged := *focus
	acknowledged.Value = "A😀中文"
	acknowledged.Selection = mirrorTextSelection{Start: 5, End: 5}
	client.applyFocusLocked(&acknowledged)
	client.updated = time.Now()
	client.applyTextSampleLocked(focus, old)
	if client.value != "A😀中文" || client.selection.Range.End != 5 {
		t.Fatal("old screenshot sample rolled back original text acknowledgement")
	}
	fresh := acknowledged
	fresh.Value = "A😀中文!"
	fresh.Selection = mirrorTextSelection{Start: 6, End: 6}
	client.applyTextSampleLocked(&fresh, time.Now().Add(time.Millisecond))
	if client.value != fresh.Value {
		t.Fatal("new original source focus sample never refreshed")
	}
}
