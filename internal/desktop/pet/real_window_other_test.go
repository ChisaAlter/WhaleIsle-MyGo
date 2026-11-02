//go:build !windows

package pet

import (
	"github.com/egoist/mygo"
	"testing"
)

func realPetMouseStyle(t *testing.T, w *mygo.Window, wantIgnoring bool) {}
