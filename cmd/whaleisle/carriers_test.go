package main

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"whaleisle/internal/ui/i18n"
)

// rawOK marks a line whose error text belongs somewhere other than copy: a
// crash record, a diagnostic log. Everything else in this package that reaches
// a person goes through internal/errcopy.
const rawOK = "raw-ok:"

type cmdRawErrorFinding struct {
	file string
	line int
	rule string
	text string
}

func scanCmdRawErrors(name string, src []byte) ([]cmdRawErrorFinding, error) {
	lines := strings.Split(string(src), "\n")
	lineHasRawOK := func(lineNo int) bool {
		if lineNo < 1 || lineNo > len(lines) {
			return false
		}
		return strings.Contains(lines[lineNo-1], rawOK)
	}
	lineText := func(lineNo int) string {
		if lineNo < 1 || lineNo > len(lines) {
			return ""
		}
		return strings.TrimSpace(lines[lineNo-1])
	}

	var findings []cmdRawErrorFinding
	seen := map[int]bool{}

	// Text check preserves existing .Error() guard semantics (including lines
	// outside AST expressions such as build-tagged comments/constructs).
	for index, line := range lines {
		if !strings.Contains(line, ".Error()") {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "if ") {
			continue
		}
		if strings.Contains(line, rawOK) {
			continue
		}
		lineNo := index + 1
		seen[lineNo] = true
		findings = append(findings, cmdRawErrorFinding{
			file: name, line: lineNo, rule: "rawerr", text: strings.TrimSpace(line),
		})
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, parser.SkipObjectResolution)
	if err != nil {
		return findings, err
	}
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if !isCmdFmtSprintCall(sel) || !cmdArgsMentionRawError(call.Args) {
			return true
		}
		lineNo := fset.Position(call.Pos()).Line
		if lineHasRawOK(lineNo) || seen[lineNo] {
			return true
		}
		seen[lineNo] = true
		findings = append(findings, cmdRawErrorFinding{
			file: name, line: lineNo, rule: "fmt-rawerr", text: lineText(lineNo),
		})
		return true
	})
	return findings, nil
}

func isCmdFmtSprintCall(sel *ast.SelectorExpr) bool {
	pkg, ok := sel.X.(*ast.Ident)
	if !ok || pkg.Name != "fmt" {
		return false
	}
	switch sel.Sel.Name {
	case "Sprint", "Sprintf", "Sprintln":
		return true
	}
	return false
}

func isErrCopyCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "errcopy"
}

// cmdArgsMentionRawError reports whether a fmt.Sprint* argument list formats a
// raw error value (identifier naming an error or nested .Error() call) without
// first sanitizing it through errcopy.*.
func cmdArgsMentionRawError(args []ast.Expr) bool {
	found := false
	for _, arg := range args {
		ast.Inspect(arg, func(n ast.Node) bool {
			if found {
				return false
			}
			switch x := n.(type) {
			case *ast.CallExpr:
				if isErrCopyCall(x) {
					return false
				}
				if sel, ok := x.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Error" && len(x.Args) == 0 {
					found = true
					return false
				}
			case *ast.Ident:
				l := strings.ToLower(x.Name)
				if l == "err" || strings.HasSuffix(l, "err") || strings.HasPrefix(l, "err") {
					found = true
					return false
				}
			}
			return true
		})
	}
	return found
}

// TestCmdNeverPaintsARawError is deliberately stricter than the guards in the
// UI packages, which look for known carriers. This package is large and its
// carriers are mostly anonymous strings handed to shared helpers, so instead of
// enumerating them the guard fails on any .Error() or fmt.Sprint* formatting an
// error value unless the line is a comparison, uses errcopy, or carries a
// raw-ok: reason.
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
		findings, err := scanCmdRawErrors(name, source)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, f := range findings {
			t.Errorf("%s:%d [%s] keeps a raw error: %s", f.file, f.line, f.rule, f.text)
		}
	}
	if scanned == 0 {
		t.Fatal("scanned no source files")
	}
}

func TestCmdCarrierASTCheckCatchesFmtAndRawError(t *testing.T) {
	sample := []byte(`package main
import (
	"fmt"
	"whaleisle/internal/errcopy"
)
func f(err error, loginErr error) {
	_ = fmt.Sprintf("fail: %v", err)
	_ = fmt.Sprint(loginErr)
	_ = err.Error()
	_ = fmt.Sprintf("safe: %s", errcopy.Sentence(err))
	_ = fmt.Sprintf("diag: %v", err) // raw-ok: diagnostic only
}
`)
	findings, err := scanCmdRawErrors("sample.go", sample)
	if err != nil {
		t.Fatalf("scan sample: %v", err)
	}
	if len(findings) != 3 {
		t.Fatalf("findings = %+v, want 3 (Sprintf(err), Sprint(loginErr), err.Error())", findings)
	}
}

func TestRawErrorsNeverLeakIntoDesktopResumeOrWatchdog(t *testing.T) {
	for _, locale := range []string{"zh", "en"} {
		t.Run(locale, func(t *testing.T) {
			i18n.SetLocale(locale)
			t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

			raw := errors.New(`exec C:\secret\tool.exe: Access is denied.`)

			desktopMsg := desktopStartupRegistrationError(raw)
			if desktopMsg == "" || strings.Contains(desktopMsg, `C:\secret`) || strings.Contains(desktopMsg, "Access is denied") {
				t.Fatalf("desktopStartupRegistrationError leaked raw error: %q", desktopMsg)
			}

			watchdogMsg := nativeRecoveryRestartFailure(raw)
			if watchdogMsg == "" || strings.Contains(watchdogMsg, `C:\secret`) || strings.Contains(watchdogMsg, "Access is denied") {
				t.Fatalf("nativeRecoveryRestartFailure leaked raw error: %q", watchdogMsg)
			}
			if locale == "zh" && strings.Contains(watchdogMsg, "Restart failed") {
				t.Fatalf("watchdog restart failure stayed hardcoded English in zh locale: %q", watchdogMsg)
			}

			// Corrupt composer draft in go-resume.json must populate s.resumeError
			// with localized copy rather than raw json/path error text.
			dir := t.TempDir()
			corrupt := []byte(`{"version":1,"drafts":{"s1":"not-an-object"}}`)
			if err := os.WriteFile(dir+"/go-resume.json", corrupt, 0o600); err != nil {
				t.Fatal(err)
			}
			s := &mainState{}
			s.openResume(dir)
			s.mu.Lock()
			resumeErr := s.resumeError
			s.mu.Unlock()
			if resumeErr == "" {
				t.Fatal("expected resumeError for corrupt draft")
			}
			if strings.Contains(resumeErr, "cannot unmarshal") || strings.Contains(resumeErr, "go-resume.json") {
				t.Fatalf("resumeError leaked raw error/path: %q", resumeErr)
			}
		})
	}
}
