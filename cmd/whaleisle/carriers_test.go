package main

import (
	"os"
	"strings"
	"testing"
)

// rawOK marks a line whose error text belongs somewhere other than copy: a
// crash record, a diagnostic log. Everything else in this package that reaches
// a person goes through internal/errcopy.
const rawOK = "raw-ok:"

// TestCmdNeverPaintsARawError is deliberately stricter than the guards in the
// UI packages, which look for known carriers. This package is large and its
// carriers are mostly anonymous strings handed to shared helpers, so instead of
// enumerating them the guard fails on any .Error() at all unless the line is a
// comparison or carries a reason.
func TestCmdNeverPaintsARawError(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}
	scanned := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		scanned++
		for index, line := range strings.Split(string(source), "\n") {
			if !strings.Contains(line, ".Error()") {
				continue
			}
			if strings.HasPrefix(strings.TrimSpace(line), "if ") {
				continue
			}
			if strings.Contains(line, rawOK) {
				continue
			}
			t.Errorf("%s:%d keeps a raw error: %s", name, index+1, strings.TrimSpace(line))
		}
	}
	if scanned == 0 {
		t.Fatal("scanned no source files")
	}
}
