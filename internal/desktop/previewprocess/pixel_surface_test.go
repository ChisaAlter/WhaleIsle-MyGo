package previewprocess

import (
	"github.com/egoist/mygo/ui"
	"image"
	"testing"
)

func TestInlinePixelInputCoordinates(t *testing.T) {
	// A 728x700 frame centered in a 1000x350 seat has 318px side bars.
	v, ok := pixelInput(ui.InputEvent{Kind: ui.InputPointerDown, X: 500, Y: 175, Button: 0}, 728, 1000, 350)
	if !ok || v.X != 364 || v.Y != 350 || v.Button != 1 || !v.Down {
		t.Fatalf("center mapped incorrectly: %+v %v", v, ok)
	}
	if _, ok := pixelInput(ui.InputEvent{Kind: ui.InputPointerDown, X: 20, Y: 175, Button: 0}, 728, 1000, 350); ok {
		t.Fatal("letterbox received input")
	}
	v, ok = pixelInput(ui.InputEvent{Kind: ui.InputPointerUp, X: 1200, Y: 500, Button: 0}, 728, 1000, 350)
	if !ok || v.X != 727 || v.Y != 699 || v.Down {
		t.Fatalf("outside release lost: %+v", v)
	}
	if _, ok := pixelInput(ui.InputEvent{Kind: ui.InputText, Text: "hello"}, 728, 0, 350); ok {
		t.Fatal("unpainted seat accepted input")
	}
}

func TestInlinePixelSeatFocusAndLifetime(t *testing.T) {
	var inputs []interactiveInput
	s := &Session{done: make(chan struct{}), closeFn: func() error { return nil }, sendTheme: func(string, map[string]string) error { return nil }}
	s.pixels = &pixelSurface{width: 728, bitmap: ui.NewBitmap(image.NewRGBA(image.Rect(0, 0, 728, 700))), enqueue: func(p packet) bool {
		if p.Input != nil {
			inputs = append(inputs, *p.Input)
		}
		return true
	}}
	height := float32(350)
	viewer := ui.NewTester(func(c *ui.Context) {
		if !s.RenderPixels(c, height, "light", nil) {
			t.Error("pixel seat not selected")
		}
	}, 1000, 700)
	viewer.ClickAt(500, 175)
	viewer.Type("hello")
	if len(inputs) < 3 || inputs[0].X != 364 || inputs[0].Y != 350 || inputs[len(inputs)-1].Text != "hello" {
		t.Fatalf("native input/focus not relayed: %+v", inputs)
	}
	height = 600
	viewer.Frame()
	viewer.ClickAt(500, 300)
	last := inputs[len(inputs)-1]
	if last.X != 364 || last.Y != 350 {
		t.Fatalf("expanded seat changed document coordinates: %+v", last)
	}
	before := len(inputs)
	_ = s.Close()
	viewer.Frame()
	viewer.ClickAt(500, 300)
	viewer.Type("closed")
	if len(inputs) != before {
		t.Fatal("closed session accepted input")
	}
}

func TestInlinePixelHiddenQueueCannotRepress(t *testing.T) {
	press := packet{Kind: "poll", Input: &interactiveInput{Kind: "pointer", X: 10, Y: 10, Button: 1, Down: true}}
	suspended := false
	command, send := inlinePixelCommand(press, false, &suspended)
	if !send || command.Kind != "release" || command.Input != nil || !suspended {
		t.Fatal("hidden queued press did not release instead")
	}
	for _, queued := range []packet{press, {Kind: "poll"}, {Kind: "poll", Input: &interactiveInput{Kind: "text", Text: "stale"}}} {
		if _, send := inlinePixelCommand(queued, false, &suspended); send {
			t.Fatal("queued hidden input escaped after release")
		}
	}
	if command, send := inlinePixelCommand(packet{Kind: "theme"}, false, &suspended); !send || command.Kind != "theme" || !suspended {
		t.Fatal("theme interrupted suspension")
	}
	if command, send := inlinePixelCommand(press, true, &suspended); !send || command.Input != press.Input || suspended {
		t.Fatal("visible seat did not resume")
	}
}
