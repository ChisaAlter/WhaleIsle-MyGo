//go:build windows

package mygo

import winhooks "github.com/egoist/mygo/internal/windows"

// The TestXxx functions bridge the Windows backend's QA hooks to apps that
// embed MyGo: their own acceptance drivers cannot reach the internal
// package. They act on a real window (Window.NativeHandle) with the messages
// and UI Automation patterns real input uses.
func TestClick(hwnd uintptr, x, y float64) bool { return winhooks.TestClickSurface(hwnd, x, y) }

func TestPressKeys(keys ...byte) { winhooks.TestPressKeys(keys...) }

func TestHoldsFrames(hwnd uintptr) (interaction, sizeBurst bool) {
	return winhooks.TestHoldsFrames(hwnd)
}

type TestAccessNode struct{ Role, Label, Value string }

func TestAccessibility(hwnd uintptr) []TestAccessNode {
	nodes, _ := winhooks.TestAccessibility(hwnd)
	out := make([]TestAccessNode, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, TestAccessNode{Role: n.Role, Label: n.Label, Value: n.Value})
	}
	return out
}

func TestAccessibilityPerform(hwnd uintptr, label, action, value string) bool {
	return winhooks.TestAccessibilityPerform(hwnd, label, action, value)
}

func TestFrameCounters() (requests, paints int64) { return winhooks.TestFrameCounters() }

func TestSendChar(hwnd uintptr, r rune) bool { return winhooks.TestSendChar(hwnd, r) }
