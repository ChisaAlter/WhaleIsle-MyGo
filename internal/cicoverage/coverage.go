// Package cicoverage guards the structural contract that every Go package
// containing tests in this repository is exercised by
// .github/workflows/main-quality.yml. New test packages that are not added to
// a workflow job fail TestAllTestedPackagesCoveredByCI instead of silently
// shipping unverified.
package cicoverage

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// scanExcludedDirs are never treated as project packages at any depth.
var scanExcludedDirs = map[string]bool{
	"third_party":  true,
	"tmp":          true,
	".toolchain":   true,
	"node_modules": true,
	".git":         true,
	".devin":       true,
	".cursor":      true,
	".agents":      true,
	".claude":      true,
	".github":      true,
	".idea":        true,
	".vscode":      true,
	".t3":          true,
	"vendor":       true,
	"testdata":     true,
	"dist":         true,
	"runtime":      true,
	"reference":    true,
}

// UncoveredExemptions lists test packages that main-quality.yml deliberately
// does not run. Every entry needs a reason why the package cannot run on a
// hosted runner. An entry whose package becomes covered by the workflow, or
// whose directory stops containing test files, is stale: the guard fails so
// the table stays honest instead of quietly hiding coverage regressions.
var UncoveredExemptions = map[string]string{
	"./tools/functional-local-settings-check": "build tag functionalacceptance only; the tests require a staged desktop runtime (WHALEISLE_FUNCTIONAL_RUNTIME with bundled node and vendor/deepseek-harness) plus explicit owned roots, so they run in the manual functional-acceptance harness rather than on a hosted runner",
}

// TestPackages walks root and returns every directory holding at least one
// *_test.go file as sorted "./"-prefixed slash paths.
func TestPackages(root string) ([]string, error) {
	found := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root {
				base := d.Name()
				if scanExcludedDirs[base] || strings.HasPrefix(base, ".") {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), "_test.go") {
			rel, err := filepath.Rel(root, filepath.Dir(path))
			if err != nil {
				return err
			}
			found["./"+filepath.ToSlash(rel)] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	pkgs := make([]string, 0, len(found))
	for p := range found {
		pkgs = append(pkgs, p)
	}
	sort.Strings(pkgs)
	return pkgs, nil
}

// CoveredPatterns extracts the package arguments of every `go test` command in
// a GitHub Actions workflow document. It understands `run:` block scalars,
// `\` line continuations, full-line comments and per-step `working-directory`,
// and returns "./"-prefixed patterns; a trailing "/..." marks a subtree
// wildcard.
func CoveredPatterns(workflow []byte) map[string]bool {
	patterns := map[string]bool{}
	lines := strings.Split(string(workflow), "\n")

	workDir := ""
	for i := 0; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r")

		// Step boundary: a new "- name:"/"- uses:"/"- run:" etc. or a new
		// mapping key resets the working-directory context for inline runs.
		if isStepItem(line) {
			workDir = ""
		}
		if key, val, ok := yamlKey(line); ok {
			switch key {
			case "working-directory":
				workDir = strings.Trim(val, `"'`)
			case "run":
				var script []string
				if val == "|" || val == "|-" || val == "|+" || val == ">" || val == ">-" || val == ">+" {
					base := indent(line)
					j := i + 1
					for ; j < len(lines); j++ {
						bl := strings.TrimRight(lines[j], "\r")
						if strings.TrimSpace(bl) == "" {
							continue
						}
						if indent(bl) <= base {
							break
						}
						script = append(script, strings.TrimSpace(bl))
					}
					i = j - 1
				} else if val != "" {
					script = []string{strings.TrimSpace(val)}
				}
				for _, pat := range goTestPackageArgs(script, workDir) {
					patterns[pat] = true
				}
			}
		}
	}
	return patterns
}

// isStepItem reports whether a line starts a new YAML sequence item (a step or
// a matrix entry), which resets the working-directory context.
func isStepItem(line string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	return strings.HasPrefix(trimmed, "- ") && len(strings.TrimSpace(trimmed[2:])) > 0
}

// yamlKey returns the key and value of a simple `key: value` line.
func yamlKey(line string) (key, val string, ok bool) {
	trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimLeft(line, " \t"), "- "))
	idx := strings.Index(trimmed, ":")
	if idx < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(trimmed[:idx])
	val = strings.TrimSpace(trimmed[idx+1:])
	if strings.HasPrefix(key, "#") {
		return "", "", false
	}
	return key, val, true
}

func indent(line string) int {
	n := 0
	for _, c := range line {
		if c != ' ' {
			break
		}
		n++
	}
	return n
}

// goTestPackageArgs finds `go test` commands inside script lines and returns
// their package arguments (tokens starting with "."), resolved against the
// step's working-directory.
func goTestPackageArgs(script []string, workDir string) []string {
	var pats []string
	for i := 0; i < len(script); i++ {
		line := script[i]
		if strings.HasPrefix(line, "#") {
			continue
		}
		// Join backslash continuations into one logical line.
		for strings.HasSuffix(line, "\\") && i+1 < len(script) {
			i++
			next := script[i]
			if strings.HasPrefix(next, "#") {
				continue
			}
			line = strings.TrimSuffix(line, "\\") + " " + next
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		for j := 0; j+1 < len(fields); j++ {
			if fields[j] != "go" || fields[j+1] != "test" {
				continue
			}
			cmdEnd := len(fields)
			for k := j + 2; k < len(fields); k++ {
				tok := fields[k]
				if tok == "&&" || tok == "||" || tok == ";" || tok == "|" || strings.HasPrefix(tok, "#") {
					cmdEnd = k
					break
				}
				if strings.HasSuffix(tok, ";") {
					cmdEnd = k + 1
					break
				}
			}
			cmdTokens := fields[j+2 : cmdEnd]
			subsetOnly := false
			for _, tok := range cmdTokens {
				if tok == "-run" || tok == "--run" || strings.HasPrefix(tok, "-run=") || strings.HasPrefix(tok, "--run=") {
					subsetOnly = true
					break
				}
			}
			if !subsetOnly {
				for _, tok := range cmdTokens {
					clean := strings.TrimSuffix(tok, ";")
					if !strings.HasPrefix(clean, ".") {
						continue
					}
					pats = append(pats, resolvePattern(clean, workDir))
				}
			}
			j = cmdEnd - 1
		}
	}
	return pats
}

// resolvePattern normalizes a package argument to a "./"-prefixed
// repo-relative pattern. A bare "." or "./x" inside a step with
// working-directory resolves relative to that directory.
func resolvePattern(tok, workDir string) string {
	tok = strings.Trim(tok, `"'`)
	wd := strings.Trim(strings.ReplaceAll(workDir, "\\", "/"), "/")
	var p string
	switch {
	case tok == ".":
		p = wd
	case strings.HasPrefix(tok, "./"):
		rest := strings.TrimPrefix(tok, "./")
		if wd != "" {
			p = wd + "/" + rest
		} else {
			p = rest
		}
	default:
		p = tok
	}
	p = strings.TrimRight(p, "/")
	if p == "" {
		return "./"
	}
	return "./" + p
}

// Covers reports whether pkg ("./a/b") matches a workflow pattern: an exact
// match, or a "./a/..." wildcard covering the directory and everything below.
func Covers(pattern, pkg string) bool {
	if pattern == pkg {
		return true
	}
	if strings.HasSuffix(pattern, "/...") {
		prefix := strings.TrimSuffix(pattern, "/...")
		return pkg == prefix || strings.HasPrefix(pkg, prefix+"/")
	}
	return false
}

// MissingCoverage returns the sorted test packages not covered by any
// workflow pattern and not listed in exemptions.
func MissingCoverage(pkgs []string, patterns map[string]bool, exemptions map[string]string) []string {
	var missing []string
	for _, p := range pkgs {
		if _, exempt := exemptions[p]; exempt {
			continue
		}
		covered := false
		for pat := range patterns {
			if Covers(pat, p) {
				covered = true
				break
			}
		}
		if !covered {
			missing = append(missing, p)
		}
	}
	sort.Strings(missing)
	return missing
}

// StaleExemptions returns exemption entries that no longer mean anything: the
// package is now covered by the workflow, or it no longer contains tests.
func StaleExemptions(pkgs []string, patterns map[string]bool, exemptions map[string]string) []string {
	pkgSet := map[string]bool{}
	for _, p := range pkgs {
		pkgSet[p] = true
	}
	var stale []string
	for p := range exemptions {
		switch {
		case !pkgSet[p]:
			stale = append(stale, p+" (no test files — delete the exemption)")
		default:
			for pat := range patterns {
				if Covers(pat, p) {
					stale = append(stale, p+" (now covered — delete the exemption)")
					break
				}
			}
		}
	}
	sort.Strings(stale)
	return stale
}
