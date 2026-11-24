// Package cicoverage guards the structural contract that every Go package
// containing tests in this repository is exercised by
// .github/workflows/main-quality.yml. New test packages that are not added to
// a workflow job fail TestAllTestedPackagesCoveredByCI instead of silently
// shipping unverified.
package cicoverage

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
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
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root {
				base := d.Name()
				if scanExcludedDirs[base] || strings.HasPrefix(base, ".") {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), "_test.go") {
			rel, err := filepath.Rel(root, filepath.Dir(p))
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

type workflowDoc struct {
	Defaults workflowDefaults       `yaml:"defaults"`
	Jobs     map[string]workflowJob `yaml:"jobs"`
}

type workflowDefaults struct {
	Run runDefaults `yaml:"run"`
}

type runDefaults struct {
	WorkingDirectory string `yaml:"working-directory"`
}

type workflowJob struct {
	If       yaml.Node        `yaml:"if"`
	Defaults workflowDefaults `yaml:"defaults"`
	Steps    []workflowStep   `yaml:"steps"`
}

type workflowStep struct {
	If               yaml.Node `yaml:"if"`
	Run              string    `yaml:"run"`
	WorkingDirectory string    `yaml:"working-directory"`
}

// ParseCoveredPatterns parses workflow with YAML semantics and extracts the
// package arguments of every full-package `go test` command in active jobs and
// steps. It returns "./"-prefixed repo-relative patterns; a trailing "/..."
// marks a subtree wildcard.
func ParseCoveredPatterns(workflow []byte) (map[string]bool, error) {
	var doc workflowDoc
	if err := yaml.Unmarshal(workflow, &doc); err != nil {
		return nil, err
	}
	patterns := map[string]bool{}
	for _, job := range doc.Jobs {
		if isDisabledIf(job.If) {
			continue
		}
		jobWD := job.Defaults.Run.WorkingDirectory
		if jobWD == "" {
			jobWD = doc.Defaults.Run.WorkingDirectory
		}
		for _, step := range job.Steps {
			if isDisabledIf(step.If) || strings.TrimSpace(step.Run) == "" {
				continue
			}
			wd := step.WorkingDirectory
			if wd == "" {
				wd = jobWD
			}
			for _, pat := range goTestPackageArgs(step.Run, wd) {
				patterns[pat] = true
			}
		}
	}
	return patterns, nil
}

// CoveredPatterns extracts the package arguments of every full-package `go test`
// command in a GitHub Actions workflow document, failing closed (empty map) if
// workflow is malformed YAML.
func CoveredPatterns(workflow []byte) map[string]bool {
	patterns, err := ParseCoveredPatterns(workflow)
	if err != nil {
		return map[string]bool{}
	}
	return patterns
}

func isDisabledIf(node yaml.Node) bool {
	if node.Kind == yaml.AliasNode && node.Alias != nil {
		node = *node.Alias
	}
	if node.Kind != yaml.ScalarNode {
		return false
	}
	s := strings.TrimSpace(node.Value)
	if strings.HasPrefix(s, "${{") && strings.HasSuffix(s, "}}") {
		s = strings.TrimSpace(s[3 : len(s)-2])
	}
	s = strings.ToLower(s)
	return s == "false" || s == "0"
}

type shellToken struct {
	val         string
	hasCmdSubst bool
}

// goTestValueFlags enumerates `go test` / `go build` flags that take a separate
// value argument when not written as `-flag=value`. Boolean flags (a, c, i, n,
// race, cover, short, v, work, etc.) must stay absent so the next package
// argument is not mistakenly consumed as a flag value.
var goTestValueFlags = map[string]bool{
	"p":                    true,
	"asmflags":             true,
	"buildmode":            true,
	"buildvcs":             true,
	"compiler":             true,
	"gccgoflags":           true,
	"gcflags":              true,
	"installsuffix":        true,
	"ldflags":              true,
	"mod":                  true,
	"modfile":              true,
	"overlay":              true,
	"pgo":                  true,
	"pkgdir":               true,
	"tags":                 true,
	"toolexec":             true,
	"o":                    true,
	"exec":                 true,
	"vet":                  true,
	"bench":                true,
	"benchtime":            true,
	"blockprofile":         true,
	"blockprofilerate":     true,
	"count":                true,
	"covermode":            true,
	"coverpkg":             true,
	"coverprofile":         true,
	"cpu":                  true,
	"cpuprofile":           true,
	"fuzz":                 true,
	"fuzzcachedir":         true,
	"fuzzminimizetime":     true,
	"fuzztime":             true,
	"list":                 true,
	"memprofile":           true,
	"memprofilerate":       true,
	"mutexprofile":         true,
	"mutexprofilerate":     true,
	"mutexprofilefraction": true,
	"outputdir":            true,
	"parallel":             true,
	"run":                  true,
	"shuffle":              true,
	"skip":                 true,
	"timeout":              true,
	"trace":                true,
}

// goTestPackageArgs finds full-package `go test` commands inside a step script
// and returns their package arguments resolved against the step's
// working-directory.
func goTestPackageArgs(runScript, workDir string) []string {
	var pats []string
	for _, line := range joinShellLines(runScript) {
		for _, cmd := range splitShellCommands(line) {
			pats = append(pats, parseGoTestCommand(cmd, workDir)...)
		}
	}
	return pats
}

func joinShellLines(runScript string) []string {
	norm := strings.ReplaceAll(runScript, "\r\n", "\n")
	raw := strings.Split(norm, "\n")
	filtered := make([]string, 0, len(raw))
	for _, l := range raw {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		filtered = append(filtered, l)
	}
	var out []string
	for i := 0; i < len(filtered); i++ {
		cur := strings.TrimRight(filtered[i], " \t")
		for strings.HasSuffix(cur, "\\") && !strings.HasSuffix(cur, "\\\\") && i+1 < len(filtered) {
			cur = strings.TrimSuffix(cur, "\\")
			i++
			cur = strings.TrimRight(cur, " \t") + " " + strings.TrimSpace(filtered[i])
		}
		out = append(out, cur)
	}
	return out
}

func splitShellCommands(line string) [][]shellToken {
	var cmds [][]shellToken
	var curCmd []shellToken
	var buf strings.Builder
	inToken := false
	hasCmdSubst := false
	inSingle := false
	inDouble := false

	flushToken := func() {
		if inToken {
			curCmd = append(curCmd, shellToken{val: buf.String(), hasCmdSubst: hasCmdSubst})
			buf.Reset()
			inToken = false
			hasCmdSubst = false
		}
	}
	flushCmd := func() {
		flushToken()
		if len(curCmd) > 0 {
			cmds = append(cmds, curCmd)
			curCmd = nil
		}
	}

	for i := 0; i < len(line); i++ {
		ch := line[i]
		if inSingle {
			if ch == '\'' {
				inSingle = false
			} else {
				buf.WriteByte(ch)
			}
			continue
		}
		if ch == '\\' && i+1 < len(line) {
			inToken = true
			next := line[i+1]
			if !inDouble || next == '"' || next == '\\' || next == '$' || next == '`' {
				buf.WriteByte(next)
				i++
				continue
			}
			buf.WriteByte(ch)
			continue
		}
		if ch == '$' && i+1 < len(line) && line[i+1] == '(' {
			inToken = true
			hasCmdSubst = true
			end := scanParenSubst(line, i+2)
			buf.WriteString(line[i:end])
			i = end - 1
			continue
		}
		if ch == '`' {
			inToken = true
			hasCmdSubst = true
			end := scanBacktickSubst(line, i+1)
			buf.WriteString(line[i:end])
			i = end - 1
			continue
		}
		if inDouble {
			if ch == '"' {
				inDouble = false
			} else {
				buf.WriteByte(ch)
			}
			continue
		}
		switch ch {
		case ' ', '\t':
			flushToken()
		case '\'':
			inSingle = true
			inToken = true
		case '"':
			inDouble = true
			inToken = true
		case '#':
			if !inToken {
				flushCmd()
				return cmds
			}
			buf.WriteByte(ch)
		case ';', '|', '&':
			flushCmd()
			if i+1 < len(line) && (ch == '&' || ch == '|') && line[i+1] == ch {
				i++
			}
		default:
			inToken = true
			buf.WriteByte(ch)
		}
	}
	flushCmd()
	return cmds
}

func scanParenSubst(s string, start int) int {
	depth := 1
	inSingle := false
	inDouble := false
	for i := start; i < len(s); i++ {
		ch := s[i]
		if inSingle {
			if ch == '\'' {
				inSingle = false
			}
			continue
		}
		if ch == '\\' && i+1 < len(s) {
			i++
			continue
		}
		if ch == '\'' && !inDouble {
			inSingle = true
			continue
		}
		if ch == '"' {
			inDouble = !inDouble
			continue
		}
		if !inDouble {
			if ch == '(' {
				depth++
			} else if ch == ')' {
				depth--
				if depth == 0 {
					return i + 1
				}
			}
		}
	}
	return len(s)
}

func scanBacktickSubst(s string, start int) int {
	for i := start; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
			continue
		}
		if s[i] == '`' {
			return i + 1
		}
	}
	return len(s)
}

func parseGoTestCommand(cmd []shellToken, workDir string) []string {
	idx := 0
	for idx < len(cmd) {
		v := cmd[idx].val
		if v == "then" || v == "do" || v == "else" || v == "elif" || v == "!" || isShellEnvAssign(v) {
			idx++
			continue
		}
		break
	}
	if idx+1 >= len(cmd) || cmd[idx].val != "go" || cmd[idx+1].val != "test" {
		return nil
	}
	args := cmd[idx+2:]
	for _, tok := range args {
		if tok.hasCmdSubst {
			return nil
		}
	}

	var pats []string
	afterDoubleDash := false
	for i := 0; i < len(args); i++ {
		tok := args[i].val
		if !afterDoubleDash {
			if tok == "--" {
				afterDoubleDash = true
				continue
			}
			if tok == "-args" || tok == "--args" {
				break
			}
			if strings.HasPrefix(tok, "-") && tok != "-" {
				flagPart := strings.TrimPrefix(strings.TrimPrefix(tok, "-"), "-")
				name, val, hasEq := strings.Cut(flagPart, "=")
				name = strings.TrimPrefix(name, "test.")
				if goTestValueFlags[name] && !hasEq {
					if i+1 >= len(args) {
						return nil
					}
					i++
					val = args[i].val
				}
				switch name {
				case "run", "list":
					return nil
				case "c", "n":
					if !(hasEq && strings.EqualFold(val, "false")) {
						return nil
					}
				case "skip":
					if isBroadSkipPattern(val) {
						return nil
					}
				}
				continue
			}
		}
		if tok == "." || strings.HasPrefix(tok, "./") {
			if pat := resolvePattern(tok, workDir); pat != "" {
				pats = append(pats, pat)
			}
		}
	}
	return pats
}

func isShellEnvAssign(tok string) bool {
	if tok == "" || strings.HasPrefix(tok, "-") {
		return false
	}
	eq := strings.IndexByte(tok, '=')
	if eq <= 0 {
		return false
	}
	name := tok[:eq]
	for i := 0; i < len(name); i++ {
		c := name[i]
		isLetter := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
		isDigit := c >= '0' && c <= '9'
		if i == 0 && !isLetter {
			return false
		}
		if !isLetter && !isDigit {
			return false
		}
	}
	return true
}

func isBroadSkipPattern(pat string) bool {
	if pat == "" {
		return false
	}
	re, err := regexp.Compile(pat)
	if err != nil {
		return true
	}
	return re.MatchString("TestCicoverageSentinelAlpha") || re.MatchString("TestUnrelatedBehaviorBeta")
}

// resolvePattern normalizes a package argument to a "./"-prefixed
// repo-relative pattern. A bare "." or "./x" inside a step with
// working-directory resolves relative to that directory.
func resolvePattern(tok, workDir string) string {
	wd := strings.TrimSpace(strings.ReplaceAll(workDir, "\\", "/"))
	if wd != "" {
		wd = path.Clean(wd)
		wd = strings.TrimPrefix(wd, "./")
		wd = strings.Trim(wd, "/")
		if wd == "." {
			wd = ""
		}
	}
	if wd == "" {
		if tok == "." {
			return "./"
		}
		cleaned := path.Clean(tok)
		if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
			return ""
		}
		if cleaned == "." {
			return "./"
		}
		return "./" + cleaned
	}
	joined := path.Join(wd, tok)
	if joined == ".." || strings.HasPrefix(joined, "../") {
		return ""
	}
	if joined == "." {
		return "./"
	}
	return "./" + joined
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
