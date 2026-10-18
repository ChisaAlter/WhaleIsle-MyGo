package previewprocess

import (
	"bytes"
	"image"
	"image/png"
	"strings"
	"testing"
)

func TestInteractiveWireBounds(t *testing.T) {
	nonce := strings.Repeat("a", 64)
	var body, wire bytes.Buffer
	_ = png.Encode(&body, image.NewRGBA(image.Rect(0, 0, 240, 700)))
	if err := sendFrame(&wire, nonce, body.Bytes(), 240); err != nil {
		t.Fatal(err)
	}
	if _, err := receiveFrame(&wire, nonce, 240); err != nil {
		t.Fatal(err)
	}
	if err := sendFrame(&wire, nonce, body.Bytes(), 241); err == nil {
		t.Fatal("accepted mismatched frame")
	}
	for _, v := range []interactiveInput{{Kind: "text", Text: strings.Repeat("x", 1025)}, {Kind: "pointer", X: 240, Button: 1}, {Kind: "key", Key: "Return exec sh"}, {Kind: "exec", Text: "sh"}} {
		if validateInteractiveInput(v, 240) == nil {
			t.Fatalf("accepted unsafe input: %+v", v)
		}
	}
	if err := validateInteractiveInput(interactiveInput{Kind: "pointer", X: 239, Y: 699, Button: 1}, 240); err != nil {
		t.Fatal(err)
	}
}

func TestInteractiveMotionBackpressurePreservesTransitions(t *testing.T) {
	commands := make(chan packet, 3)
	motion := packet{Kind: "poll", Input: &interactiveInput{Kind: "pointer", X: 10}}
	press := packet{Kind: "poll", Input: &interactiveInput{Kind: "pointer", X: 20, Button: 1, Down: true}}
	release := packet{Kind: "poll", Input: &interactiveInput{Kind: "pointer", X: 30, Button: 1}}
	if !enqueueInteractiveCommand(commands, motion) || !enqueueInteractiveCommand(commands, press) {
		t.Fatal("initial commands rejected")
	}
	for i := 0; i < 1000; i++ {
		if !enqueueInteractiveCommand(commands, motion) {
			t.Fatal("motion flood closed viewer")
		}
	}
	if len(commands) != 2 {
		t.Fatal("motion flood accumulated")
	}
	if !enqueueInteractiveCommand(commands, release) {
		t.Fatal("motion consumed release capacity")
	}
	for _, want := range []packet{motion, press, release} {
		got := <-commands
		if *got.Input != *want.Input {
			t.Fatalf("input order changed: %+v", got.Input)
		}
	}
	commands <- press
	commands <- release
	commands <- press
	if enqueueInteractiveCommand(commands, release) {
		t.Fatal("critical overflow silently accepted")
	}
	if !enqueueInteractiveCommand(commands, motion) {
		t.Fatal("full queue rejected disposable motion")
	}
}
