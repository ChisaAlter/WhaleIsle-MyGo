package imagepreview

import (
	"context"
	"errors"
	"testing"
)

func TestNativeDecodersCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := DecodeSVGContext(ctx, []byte(`<svg width="10" height="10"/>`)); !errors.Is(err, context.Canceled) {
		t.Fatalf("SVG lost cancellation: %v", err)
	}
	if _, err := DecodeAVIFContext(ctx, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("AVIF lost cancellation: %v", err)
	}
}
