//go:build !windows

package ui

import (
	"image"
	"time"
)

type cpuSplashBackend struct{ solver *splashSolver }

func newSplashBackend() (splashBackend, error)      { return &cpuSplashBackend{}, nil }
func nativeSplashFits(width, height int) bool       { return cpuSplashFits(width, height) }
func splashPacing(work time.Duration) time.Duration { return cpuSplashPacing(work) }
func (b *cpuSplashBackend) Resize(w, h int) error {
	if b.solver == nil {
		b.solver = newSplashSolver(w, h, nativeSplashSimResolution, nativeSplashDyeResolution)
	} else {
		b.solver.Resize(w, h)
	}
	return nil
}
func (b *cpuSplashBackend) Splat(x, y, dx, dy float32, color [3]float32, size float32) {
	b.solver.Splat(x, y, dx, dy, color, size)
}
func (b *cpuSplashBackend) Step(dt, speed float32)       { b.solver.Step(dt, speed) }
func (b *cpuSplashBackend) ImageSize() (int, int)        { return b.solver.ImageSize() }
func (b *cpuSplashBackend) Render(dst *image.RGBA) error { b.solver.Render(dst, true); return nil }
func (b *cpuSplashBackend) Close()                       { b.solver = nil }
