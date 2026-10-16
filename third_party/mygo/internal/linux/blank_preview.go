//go:build linux && (amd64 || arm64)

package linux

import "errors"

var blankContextNew func() ptr
var blankContextIsEphemeral func(ptr) bool

func loadBlankPreviewAPI(lib ptr) {
	bind(lib, &blankContextNew, "webkit_web_context_new_ephemeral")
	bind(lib, &blankContextIsEphemeral, "webkit_web_context_is_ephemeral")
}
func newBlankPreviewContext() (ptr, error) {
	if blankContextNew == nil || blankContextIsEphemeral == nil || browserObjectNew == nil || browserWebViewType == nil {
		return 0, errors.New("mygo: private blank encoder requires WebKitGTK ephemeral-context APIs")
	}
	ctx := blankContextNew()
	if ctx == 0 {
		return 0, errors.New("mygo: cannot create private blank encoder context")
	}
	if !blankContextIsEphemeral(ctx) {
		gObjectUnref(ctx)
		return 0, errors.New("mygo: blank encoder context is not ephemeral")
	}
	return ctx, nil
}
