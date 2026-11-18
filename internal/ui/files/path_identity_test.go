package files

import (
	"path/filepath"
	"strings"
)

// samePath reports whether two names refer to the same location. Hosted Windows
// runners hand out temp directories through short names and junctions, so one
// file can legitimately arrive spelled two ways; these tests care about which
// file was acted on, not about how the temp root happens to be spelled.
func samePath(a, b string) bool {
	if a == b {
		return true
	}
	cleanA, errA := filepath.Abs(a)
	cleanB, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return false
	}
	if strings.EqualFold(cleanA, cleanB) {
		return true
	}
	resolvedA, errA := filepath.EvalSymlinks(cleanA)
	resolvedB, errB := filepath.EvalSymlinks(cleanB)
	return errA == nil && errB == nil && strings.EqualFold(resolvedA, resolvedB)
}
