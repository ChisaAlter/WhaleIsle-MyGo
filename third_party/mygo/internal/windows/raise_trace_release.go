//go:build windows && (amd64 || arm64) && release

package windows

func traceRaise(string, uintptr) {}

func activationCause(uint16) string { return "" }
