//go:build windows && release

package windows

func recordTestFrameRequest()         {}
func recordTestPaintMsg()             {}
func consumeTestWebViewFailure() bool { return false }
func TestFailWebViews(int)            {}
