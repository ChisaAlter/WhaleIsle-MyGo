//go:build linux && (amd64 || arm64)

package linux

import (
	"github.com/egoist/mygo/internal/platform"
	"testing"
)

func TestBrowserSeatClipRect(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	cases := []struct {
		in   platform.Rect
		want gdkRectangle
		ok   bool
	}{
		{platform.Rect{X: -10, Y: 20, Width: 30, Height: 100}, gdkRectangle{X: 0, Y: 20, Width: 20, Height: 80}, true},
		{platform.Rect{X: 150, Y: 10, Width: 90, Height: 20}, gdkRectangle{X: 150, Y: 10, Width: 50, Height: 20}, true},
		{platform.Rect{X: maxInt - 1, Y: 0, Width: 100, Height: 20}, gdkRectangle{}, false},
		{platform.Rect{X: -maxInt, Y: 0, Width: maxInt, Height: 20}, gdkRectangle{}, false},
		{platform.Rect{X: 1, Y: 1, Width: maxInt, Height: maxInt}, gdkRectangle{X: 1, Y: 1, Width: 199, Height: 99}, true},
		{platform.Rect{Width: -1, Height: 20}, gdkRectangle{}, false},
	}
	for _, tc := range cases {
		got, ok := browserSeatClipRect(200, 100, tc.in)
		if got != tc.want || ok != tc.ok {
			t.Fatalf("%+v: got %+v %v, want %+v %v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}
