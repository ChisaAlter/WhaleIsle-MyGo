//go:build darwin && amd64

package voicecapture

import "github.com/ebitengine/purego/objc"

// objc.h defines BOOL as signed char on macOS x86_64 (OBJC_BOOL_IS_BOOL=0).
func newMacPermissionBlock(done func(bool)) objc.Block {
	return objc.NewBlock(func(_ objc.Block, granted int8) { done(granted != 0) })
}
