package imagepreview

import (
	"context"
	"errors"
	"testing"
)

func TestNativeDecodersCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := DecodeAVIFContext(ctx, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("AVIF lost cancellation: %v", err)
	}
}
