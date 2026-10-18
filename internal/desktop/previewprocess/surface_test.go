package previewprocess

import (
	"bytes"
	"strings"
	"testing"
)

func TestInlineDocumentProtocolPreservesPresentationMode(t *testing.T) {
	d := Document{HTML: "<button>stateful</button>", Width: 728, Appearance: "light", Inline: true}
	var wire bytes.Buffer
	nonce := strings.Repeat("a", 64)
	if err := sendDocument(&wire, nonce, d); err != nil {
		t.Fatal(err)
	}
	got, err := receiveDocument(&wire, nonce)
	if err != nil || got != d {
		t.Fatalf("inline document: %+v %v", got, err)
	}
}
func TestLiveThemeCommandsAreBoundedAndDeduplicated(t *testing.T) {
	var themes []string
	s := &Session{done: make(chan struct{}), sendTheme: func(v string, _ map[string]string) error { themes = append(themes, v); return nil }}
	for _, v := range []string{"light", "dark", "dark", "light"} {
		if err := s.SetAppearance(v); err != nil {
			t.Fatal(err)
		}
	}
	if strings.Join(themes, ",") != "light,dark,light" {
		t.Fatalf("unexpected updates: %v", themes)
	}
	if s.SetAppearance("dark;document.open()") == nil {
		t.Fatal("accepted arbitrary theme command")
	}
}
