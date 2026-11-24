package composer

import (
	"context"
	"encoding/json"
	"image"
	"image/color"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

func TestReferenceQueryDebouncesAndRetainsPriorRowsInFlight(t *testing.T) {
	d := NewDraftState("")
	var calls atomic.Int32
	release := make(chan struct{})
	done := make(chan struct{}, 4)

	m := Model{
		State:    d,
		Snapshot: session.ViewState{SelectedID: "s1", ComposerHostGeneration: 1},
		OnQueryReferences: func(ctx context.Context, q string, _ bool) ([]session.ReferenceCandidate, error) {
			calls.Add(1)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-release:
				return []session.ReferenceCandidate{{Kind: "file", Path: q + ".go"}}, nil
			}
		},
		referenceInvalidate: func() {
			select {
			case done <- struct{}{}:
			default:
			}
		},
	}

	// First query completes and populates candidate rows.
	close(release)
	d.references.query(m, "main", false)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("initial reference query did not complete")
	}
	rows := d.references.query(m, "main", false)
	if len(rows) != 1 || rows[0].Path != "main.go" {
		t.Fatalf("initial rows = %+v", rows)
	}

	// Rapid keystrokes must debounce (not fire immediately) and keep prior rows
	// visible while the replacement query is pending (no empty flash).
	block := make(chan struct{})
	m.OnQueryReferences = func(ctx context.Context, q string, _ bool) ([]session.ReferenceCandidate, error) {
		calls.Add(1)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-block:
			return []session.ReferenceCandidate{{Kind: "file", Path: q + ".go"}}, nil
		}
	}
	beforeCalls := calls.Load()
	retained1 := d.references.query(m, "main_", false)
	retained2 := d.references.query(m, "main_t", false)
	if len(retained1) != 1 || retained1[0].Path != "main.go" || len(retained2) != 1 || retained2[0].Path != "main.go" {
		t.Fatalf("expected prior rows retained while debounced query is in flight, got %+v / %+v", retained1, retained2)
	}
	time.Sleep(40 * time.Millisecond)
	if got := calls.Load(); got != beforeCalls {
		t.Fatalf("debounced query fired too early: calls=%d before=%d", got, beforeCalls)
	}
	close(block)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("debounced replacement query did not finish")
	}
	finalRows := d.references.query(m, "main_t", false)
	if len(finalRows) != 1 || finalRows[0].Path != "main_t.go" {
		t.Fatalf("final rows = %+v, want main_t.go", finalRows)
	}
	if calls.Load() != beforeCalls+1 {
		t.Fatalf("expected rapid keystrokes to coalesce into 1 RPC, got %d", calls.Load()-beforeCalls)
	}
	d.references.cancelQuery()
}

func BenchmarkComposerViewIdle(b *testing.B) {
	draft := "Summarize the architecture of internal/ui/composer and check https://example.com/docs for reference."
	d := NewDraftState(draft)
	beamRaw := json.RawMessage(`{"direction":"clockwise","period":2.0,"intensity":100}`)
	typingRaw := json.RawMessage(`{"effect":"drop","cursor":"native","cursorBlink":true,"speed":100}`)
	m := Model{
		Draft:             &d.Text,
		State:             d,
		TextInput:         d.Client,
		ThemeID:           "deepseek.light",
		ComposerBeam:      false,
		ComposerBeamStyle: beamRaw,
		TypingFx:          false,
		TypingFxStyle:     typingRaw,
		OnOpenURL:         func(string) {},
		Snapshot: session.ViewState{
			SelectedID:  "bench-s",
			ClientReady: true,
		},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { View(c, m) })
	}, 720, 240)
	tt.Frame()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tt.Frame()
	}
}

func BenchmarkProjectModelFlip(b *testing.B) {
	src := image.NewRGBA(image.Rect(0, 0, 240, 32))
	for y := 6; y < 26; y++ {
		for x := 8; x < 200; x++ {
			src.SetRGBA(x, y, color.RGBA{R: 220, G: 220, B: 230, A: 255})
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dst, _ := projectModelFlip(src, 120, 16, 2, 42, 0.85)
		if dst == nil {
			b.Fatal("nil projection")
		}
	}
}
