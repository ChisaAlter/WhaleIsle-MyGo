package svgimage

import (
	"bytes"
	"context"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func TestSeatRetainsBoundedOriginalSnapshot(t *testing.T) {
	data := []byte(`<svg width="1in" height=".5in"><animate attributeName="opacity" dur="1s" values="0;1"/></svg>`)
	original := bytes.Clone(data)
	s, err := NewSeat(data)
	if err != nil {
		t.Fatal(err)
	}
	data[0] = 'x'
	if !bytes.Equal(s.data, original) {
		t.Fatal("caller mutated the seat's authorized snapshot")
	}
	if w, h := s.Size(); w != 96 || h != 48 {
		t.Fatalf("unexpected intrinsic hint %vx%v", w, h)
	}
	s.Close()
	s.Close()
}

func TestSeatRejectsAbsentOrRevokedOwnerBeforeHelper(t *testing.T) {
	for _, owner := range []*Owner{nil, {}, {ID: "revoked", Window: &mygo.Window{}, Context: context.Background(), HelperPath: "must-not-start", Valid: func() bool { return false }}} {
		s, err := NewSeat([]byte(`<svg width="10em" height="5ex"/>`))
		if err != nil {
			t.Fatal(err)
		}
		ui.NewTester(func(c *ui.Context) { s.Prepare(c, owner) }, 300, 150)
		if s.Error() == nil || s.pending || s.surface != nil {
			t.Fatal("invalid owner acquired a helper", s.Error())
		}
		s.Close()
	}
}
