//go:build windows && (amd64 || arm64)

package windows

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestContentOuterPixelsPreservesExactClientSizeAndOrigin(t *testing.T) {
	for _, dpi := range []int{96, 120, 144, 192} {
		for _, insets := range []rect{{}, {Top: 1}, {Left: 8, Top: 1, Right: 8, Bottom: 8}, {Left: 8, Top: 31, Right: 8, Bottom: 8}} {
			t.Run(fmt.Sprintf("dpi%d_insets%v", dpi, insets), func(t *testing.T) {
				// Odd and negative DIP origins ensure far-edge rounding cannot
				// change width/height or creep the client origin on repeated calls.
				for _, origin := range []int{-101, 1, 3, 101} {
					request := platform.Rect{X: origin, Y: origin + 2, Width: 161, Height: 121}
					outer := contentOuterPixels(request, dpi, insets)
					client := rect{outer.Left + insets.Left, outer.Top + insets.Top, outer.Right - insets.Right, outer.Bottom - insets.Bottom}
					if client.Left != toPx(request.X, dpi) || client.Top != toPx(request.Y, dpi) || client.Right-client.Left != toPx(request.Width, dpi) || client.Bottom-client.Top != toPx(request.Height, dpi) {
						t.Fatalf("requested=%+v outer=%+v actualclient=%+v", request, outer, client)
					}
					if got := contentOuterPixels(request, dpi, insets); got != outer {
						t.Fatal("repeated content resize changed outer origin")
					}
				}
			})
		}
	}
}
