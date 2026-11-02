//go:build darwin && arm64

package voicecapture

import "github.com/ebitengine/purego/objc"

// objc.h defines BOOL as bool on macOS arm64 (OBJC_BOOL_IS_BOOL=1).
func newMacPermissionBlock(done func(bool)) objc.Block {
	return objc.NewBlock(func(_ objc.Block, granted bool) { done(granted) })
}
