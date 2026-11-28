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
	"strconv"
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
	Env      map[string]string      `yaml:"env"`
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
	If              yaml.Node         `yaml:"if"`
	ContinueOnError yaml.Node         `yaml:"continue-on-error"`
	Env             map[string]string `yaml:"env"`
	Defaults        workflowDefaults  `yaml:"defaults"`
	Steps           []workflowStep    `yaml:"steps"`
}

type workflowStep struct {
	If               yaml.Node         `yaml:"if"`
	ContinueOnError  yaml.Node         `yaml:"continue-on-error"`
	Env              map[string]string `yaml:"env"`
	Run              string            `yaml:"run"`
	WorkingDirectory string            `yaml:"working-directory"`
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
	docGoFlags, _ := lookupEnvCI(doc.Env, "GOFLAGS")
	patterns := map[string]bool{}
	for _, job := range doc.Jobs {
		if isDisabledIf(job.If) || isContinueOnError(job.ContinueOnError) {
			continue
		}
		jobWD := job.Defaults.Run.WorkingDirectory
		if jobWD == "" {
			jobWD = doc.Defaults.Run.WorkingDirectory
		}
		jobGoFlags := docGoFlags
		if v, ok := lookupEnvCI(job.Env, "GOFLAGS"); ok {
			jobGoFlags = v
		}
		for _, step := range job.Steps {
			if isDisabledIf(step.If) || isContinueOnError(step.ContinueOnError) || strings.TrimSpace(step.Run) == "" {
				continue
			}
			stepGoFlags := jobGoFlags
			if v, ok := lookupEnvCI(step.Env, "GOFLAGS"); ok {
				stepGoFlags = v
			}
			if strings.TrimSpace(stepGoFlags) != "" {
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

func lookupEnvCI(env map[string]string, key string) (string, bool) {
	for k, v := range env {
		if strings.EqualFold(k, key) {
			return v, true
		}
	}
	return "", false
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
	for len(s) >= 2 && strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	s = strings.ToLower(s)
	return s == "false" || s == "0" || s == "!true" || s == "!1"
}

func isContinueOnError(node yaml.Node) bool {
	if node.Kind == yaml.AliasNode && node.Alias != nil {
		node = *node.Alias
	}
	if node.Kind == 0 || node.Kind != yaml.ScalarNode {
		return false
	}
	s := strings.TrimSpace(node.Value)
	if strings.HasPrefix(s, "${{") && strings.HasSuffix(s, "}}") {
		s = strings.TrimSpace(s[3 : len(s)-2])
	}
	for len(s) >= 2 && strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	s = strings.ToLower(s)
	return s != "" && s != "false" && s != "0" && s != "!true" && s != "!1"
}

type shellToken struct {
	val         string
	hasCmdSubst bool
}

type shellCommand struct {
	tokens       []shellToken
	followedByOr bool
}

// goTestValueFlags enumerates `go test` / `go build` flags that take a separate
// value argument when not written as `-flag=value`. Boolean flags (a, c, i, n,
// race, cover, short, v, work, etc.) must stay absent so the next package
// argument is not mistakenly consumed as a flag value.
var goTestValueFlags = map[string]bool{
	"C":                    true,
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

type heredocSpec struct {
	delim     string
	stripTabs bool
}

// goTestPackageArgs finds full-package `go test` commands inside a step script
// and returns their package arguments resolved against the step's
// working-directory.
func goTestPackageArgs(runScript, workDir string) []string {
	var pats []string
	stepGoFlagsActive := false
	for _, line := range joinShellLines(runScript) {
		for _, cmd := range splitShellCommands(line) {
			if updateStepGoFlags(cmd.tokens, &stepGoFlagsActive) {
				continue
			}
			if cmd.followedByOr || stepGoFlagsActive {
				continue
			}
			pats = append(pats, parseGoTestCommand(cmd.tokens, workDir)...)
		}
	}
	return pats
}

func updateStepGoFlags(tokens []shellToken, active *bool) bool {
	if len(tokens) == 0 {
		return false
	}
	if tokens[0].val == "export" {
		for _, tok := range tokens[1:] {
			if isShellEnvAssign(tok.val) {
				k, v, _ := strings.Cut(tok.val, "=")
				if strings.EqualFold(k, "GOFLAGS") {
					*active = strings.TrimSpace(v) != ""
				}
			}
		}
		return true
	}
	if tokens[0].val == "unset" {
		for _, tok := range tokens[1:] {
			if strings.EqualFold(tok.val, "GOFLAGS") {
				*active = false
			}
		}
		return true
	}
	for _, tok := range tokens {
		if !isShellEnvAssign(tok.val) {
			return false
		}
	}
	for _, tok := range tokens {
		k, v, _ := strings.Cut(tok.val, "=")
		if strings.EqualFold(k, "GOFLAGS") {
			*active = strings.TrimSpace(v) != ""
		}
	}
	return true
}

func joinShellLines(runScript string) []string {
	norm := strings.ReplaceAll(runScript, "\r\n", "\n")
	raw := strings.Split(norm, "\n")
	var pending []heredocSpec
	filtered := make([]string, 0, len(raw))
	for i := 0; i < len(raw); i++ {
		line := raw[i]
		if len(pending) > 0 {
			cand := line
			if pending[0].stripTabs {
				cand = strings.TrimLeft(cand, "\t")
			}
			if cand == pending[0].delim {
				pending = pending[1:]
			}
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		filtered = append(filtered, line)
		cur := strings.TrimRight(line, " \t")
		for strings.HasSuffix(cur, "\\") && !strings.HasSuffix(cur, "\\\\") && i+1 < len(raw) {
			i++
			next := raw[i]
			nextTrim := strings.TrimSpace(next)
			if nextTrim != "" && !strings.HasPrefix(nextTrim, "#") {
				filtered[len(filtered)-1] = strings.TrimRight(strings.TrimSuffix(cur, "\\"), " \t") + " " + nextTrim
			}
			cur = strings.TrimRight(filtered[len(filtered)-1], " \t")
		}
		pending = append(pending, scanHeredocs(filtered[len(filtered)-1])...)
	}
	return filtered
}

func scanHeredocs(line string) []heredocSpec {
	var specs []heredocSpec
	inSingle := false
	inDouble := false
	arithDepth := 0
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if inSingle {
			if ch == '\'' {
				inSingle = false
			}
			continue
		}
		if ch == '\\' && i+1 < len(line) {
			i++
			continue
		}
		if inDouble {
			if ch == '"' {
				inDouble = false
			}
			continue
		}
		if arithDepth > 0 {
			if ch == '(' && i+1 < len(line) && line[i+1] == '(' {
				arithDepth++
				i++
			} else if ch == ')' && i+1 < len(line) && line[i+1] == ')' {
				arithDepth--
				i++
			}
			continue
		}
		switch ch {
		case '\'':
			inSingle = true
		case '"':
			inDouble = true
		case '#':
			if i == 0 || line[i-1] == ' ' || line[i-1] == '\t' || line[i-1] == ';' || line[i-1] == '&' || line[i-1] == '|' {
				return specs
			}
		case '$':
			if i+2 < len(line) && line[i+1] == '(' && line[i+2] == '(' {
				arithDepth++
				i += 2
			}
		case '(':
			if i+1 < len(line) && line[i+1] == '(' {
				arithDepth++
				i++
			}
		case '<':
			if i+1 < len(line) && line[i+1] == '<' {
				if i+2 < len(line) && line[i+2] == '<' {
					i += 2
					continue
				}
				j := i + 2
				stripTabs := false
				if j < len(line) && line[j] == '-' {
					stripTabs = true
					j++
				}
				for j < len(line) && (line[j] == ' ' || line[j] == '\t') {
					j++
				}
				delim, nextIdx := readHeredocDelim(line, j)
				if delim != "" {
					specs = append(specs, heredocSpec{delim: delim, stripTabs: stripTabs})
				}
				i = nextIdx - 1
			}
		}
	}
	return specs
}

func readHeredocDelim(line string, start int) (string, int) {
	var b strings.Builder
	inS := false
	inD := false
	i := start
	for i < len(line) {
		ch := line[i]
		if inS {
			if ch == '\'' {
				inS = false
			} else {
				b.WriteByte(ch)
			}
			i++
			continue
		}
		if inD {
			if ch == '\\' && i+1 < len(line) {
				b.WriteByte(line[i+1])
				i += 2
				continue
			}
			if ch == '"' {
				inD = false
			} else {
				b.WriteByte(ch)
			}
			i++
			continue
		}
		if ch == ' ' || ch == '\t' || ch == ';' || ch == '|' || ch == '&' || ch == '<' || ch == '>' || ch == ')' || ch == '(' {
			break
		}
		switch ch {
		case '\'':
			inS = true
		case '"':
			inD = true
		case '\\':
			if i+1 < len(line) {
				b.WriteByte(line[i+1])
				i += 2
				continue
			}
		default:
			b.WriteByte(ch)
		}
		i++
	}
	return b.String(), i
}

func splitShellCommands(line string) []shellCommand {
	var cmds []shellCommand
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
	flushCmd := func(followedByOr bool) {
		flushToken()
		if len(curCmd) > 0 {
			cmds = append(cmds, shellCommand{tokens: curCmd, followedByOr: followedByOr})
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
				flushCmd(false)
				return cmds
			}
			buf.WriteByte(ch)
		case ';', '|', '&':
			isOr := ch == '|' && i+1 < len(line) && line[i+1] == '|'
			flushCmd(isOr)
			if i+1 < len(line) && (ch == '&' || ch == '|') && line[i+1] == ch {
				i++
			}
		default:
			inToken = true
			buf.WriteByte(ch)
		}
	}
	flushCmd(false)
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
		if isShellEnvAssign(v) {
			k, val, _ := strings.Cut(v, "=")
			if strings.EqualFold(k, "GOFLAGS") && strings.TrimSpace(val) != "" {
				return nil
			}
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
	effectiveWD := workDir
	seenPkg := false
	stopPkgs := false
	for i := 0; i < len(args); i++ {
		tok := args[i].val
		if tok == "--" || tok == "-args" || tok == "--args" {
			break
		}
		if strings.HasPrefix(tok, "-") && tok != "-" {
			if seenPkg {
				stopPkgs = true
			}
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
			case "run", "list", "exec":
				return nil
			case "c", "n":
				if !(hasEq && strings.EqualFold(val, "false")) {
					return nil
				}
			case "count":
				n, err := strconv.Atoi(strings.TrimSpace(val))
				if err != nil || n <= 0 {
					return nil
				}
			case "C":
				nextWD, ok := resolveSubdir(effectiveWD, val)
				if !ok {
					return nil
				}
				effectiveWD = nextWD
			case "skip":
				if isBroadSkipPattern(val) {
					return nil
				}
			}
			continue
		}
		if !stopPkgs && (tok == "." || strings.HasPrefix(tok, "./")) {
			seenPkg = true
			if pat := resolvePattern(tok, effectiveWD); pat != "" {
				pats = append(pats, pat)
			}
		}
	}
	return pats
}

func resolveSubdir(baseWD, dirArg string) (string, bool) {
	dirArg = strings.TrimSpace(strings.ReplaceAll(dirArg, "\\", "/"))
	if dirArg == "" || strings.HasPrefix(dirArg, "/") || (len(dirArg) >= 2 && dirArg[1] == ':') {
		return "", false
	}
	wd := normalizeWorkDir(baseWD)
	var joined string
	if wd == "" {
		joined = path.Clean(dirArg)
	} else {
		joined = path.Clean(wd + "/" + dirArg)
	}
	if joined == ".." || strings.HasPrefix(joined, "../") {
		return "", false
	}
	if joined == "." {
		return "", true
	}
	return joined, true
}

func normalizeWorkDir(workDir string) string {
	wd := strings.TrimSpace(strings.ReplaceAll(workDir, "\\", "/"))
	if wd == "" {
		return ""
	}
	wd = path.Clean(wd)
	wd = strings.TrimPrefix(wd, "./")
	wd = strings.Trim(wd, "/")
	if wd == "." {
		return ""
	}
	return wd
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
	parts := strings.Split(pat, "/")
	if len(parts) > 1 {
		for _, p := range parts {
			re, err := regexp.Compile(p)
			if err != nil {
				return true
			}
			if re.MatchString("TestCicoverageSentinelAlpha") || re.MatchString("TestUnrelatedBehaviorBeta") || re.MatchString("subtest_alpha") {
				return true
			}
		}
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
	wd := normalizeWorkDir(workDir)
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

// MjsUncoveredExemptions lists *.test.mjs files that main-quality.yml
// deliberately does not run. Every entry needs the reason the file cannot run
// on a hosted runner. An entry whose file becomes wired into the workflow, or
// that stops existing, is stale: the guard fails so the table stays honest.
var MjsUncoveredExemptions = map[string]string{
	"scripts/build-storage.test.mjs":                "requires the pinned local NSIS installation for real packaging verification, which a hosted runner deliberately does not carry",
	"internal/harness/native-mcp-import.test.mjs":   "imports modules from the expanded runtime-rc4 vendor tree, which a CI checkout does not carry",
	"internal/harness/native-skill-import.test.mjs": "imports modules from the expanded runtime-rc4 vendor tree, which a CI checkout does not carry",
	"scripts/session-runtime.test.mjs":              "its clone/stage/verify cases are gated on WHALEISLE_SESSION_RUNTIME_STAGED_FIXTURE naming a verified staged runtime that only the manual runtime-acceptance harness provisions",
	"scripts/windows-background-runtime.test.mjs":   "reads the expanded runtime/vendor/dsh-remote tree, which a CI checkout does not carry",
}

// MjsTestFiles walks root and returns every *.test.mjs file as sorted slash
// paths relative to root. The same machine-noise directories as the Go scan
// are skipped, plus runtime sibling checkouts (runtime-rc4 and friends, which
// carry their own vendored test files) and the untracked bin output dir.
func MjsTestFiles(root string) ([]string, error) {
	found := map[string]bool{}
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root {
				base := d.Name()
				if scanExcludedDirs[base] || strings.HasPrefix(base, ".") || strings.HasPrefix(base, "runtime") || base == "bin" {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".test.mjs") {
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return err
			}
			found[filepath.ToSlash(rel)] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(found))
	for f := range found {
		files = append(files, f)
	}
	sort.Strings(files)
	return files, nil
}

// MissingMjsCoverage returns every file that is neither named in the workflow
// text nor exempted. Matching is textual: the workflow must reference the
// file by its repository-relative slash path. A reference that only exists
// inside a comment still counts, which the ratchet accepts because the
// failure it exists to catch is a file nobody ever wired.
func MissingMjsCoverage(files []string, workflowRaw []byte, exemptions map[string]string) []string {
	text := string(workflowRaw)
	var missing []string
	for _, f := range files {
		if _, exempt := exemptions[f]; exempt {
			continue
		}
		if !strings.Contains(text, f) {
			missing = append(missing, f)
		}
	}
	sort.Strings(missing)
	return missing
}

// StaleMjsExemptions returns exemption entries whose file no longer exists or
// is already wired into the workflow, so the table cannot hide coverage that
// became real or pin files that were deleted.
func StaleMjsExemptions(files []string, workflowRaw []byte, exemptions map[string]string) []string {
	text := string(workflowRaw)
	existing := map[string]bool{}
	for _, f := range files {
		existing[f] = true
	}
	var stale []string
	for f := range exemptions {
		if !existing[f] || strings.Contains(text, f) {
			stale = append(stale, f)
		}
	}
	sort.Strings(stale)
	return stale
}
