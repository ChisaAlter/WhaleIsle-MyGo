package previewprocess

import (
	"bytes"
	"errors"
	"image/png"
	"io"
	"strings"
	"unicode/utf8"
)

const interactiveHeight = 700
const maxFrameBytes = 8 * 1024 * 1024

type interactiveInput struct {
	Kind   string `json:"kind"`
	X      int    `json:"x,omitempty"`
	Y      int    `json:"y,omitempty"`
	Button int    `json:"button,omitempty"`
	Down   bool   `json:"down,omitempty"`
	Key    string `json:"key,omitempty"`
	Text   string `json:"text,omitempty"`
}

func validateInteractiveInput(v interactiveInput, width int) error {
	switch v.Kind {
	case "pointer":
		if v.X >= 0 && v.X < width && v.Y >= 0 && v.Y < interactiveHeight && v.Button >= 0 && v.Button <= 5 {
			return nil
		}
	case "text":
		if len(v.Text) > 0 && len(v.Text) <= 1024 && utf8.ValidString(v.Text) && !strings.ContainsRune(v.Text, 0) {
			return nil
		}
	case "key":
		switch v.Key {
		case "Return", "Escape", "BackSpace", "Tab", "Delete", "Home", "End", "Prior", "Next", "Left", "Right", "Up", "Down":
			return nil
		}
	}
	return errors.New("invalid isolated preview input")
}
func validateFrame(data []byte, width int) error {
	if len(data) == 0 || len(data) > maxFrameBytes {
		return errors.New("invalid preview frame size")
	}
	c, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return err
	}
	if c.Width != width || c.Height != interactiveHeight {
		return errors.New("invalid preview frame dimensions")
	}
	return nil
}
func sendFrame(w io.Writer, nonce string, data []byte, width int) error {
	if err := validateFrame(data, width); err != nil {
		return err
	}
	if err := writePacket(w, packet{Version: protocolVersion, Kind: "frame", Nonce: nonce, Bytes: len(data), Width: width, Height: interactiveHeight}); err != nil {
		return err
	}
	return writeAll(w, data)
}
func receiveFrame(r io.Reader, nonce string, width int) ([]byte, error) {
	p, err := readPacket(r)
	if err != nil {
		return nil, err
	}
	if err = expect(p, "frame", nonce); err != nil {
		return nil, err
	}
	if p.Bytes < 1 || p.Bytes > maxFrameBytes || p.Width != width || p.Height != interactiveHeight {
		return nil, errors.New("invalid preview frame metadata")
	}
	data := make([]byte, p.Bytes)
	if _, err = io.ReadFull(r, data); err != nil {
		return nil, err
	}
	return data, validateFrame(data, width)
}

// Keep motion bounded behind the in-flight frame. Button transitions carry their
// own coordinates, so skipping intermediate hover/drag samples never loses a
// press/release or moves a click to an old point. Non-motion input keeps FIFO.
func enqueueInteractiveCommand(commands chan packet, p packet) bool {
	motion := p.Kind == "poll" && p.Input != nil && p.Input.Kind == "pointer" && p.Input.Button == 0
	if motion && len(commands) > 0 {
		return true
	}
	select {
	case commands <- p:
		return true
	default:
		return motion
	}
}
