//go:build windows

package ui

import (
	"time"

	"github.com/egoist/mygo/internal/gpu/d3d11"
)

func newSplashBackend() (splashBackend, error) { return d3d11.NewFluid() }
func nativeSplashFits(width, height int) bool  { return d3d11.FluidFits(width, height) }
func splashPacing(time.Duration) time.Duration { return splashFrameInterval }
