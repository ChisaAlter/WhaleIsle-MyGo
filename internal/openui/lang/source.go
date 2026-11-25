package lang

import "strings"

// Limits bound the work done on untrusted model output. Zero fields take the
// defaults of DefaultLimits. The defaults equal the Whale UI viewer runtime
// budgets (viewer/v2, `Xt`), with the program size bounded by the wire limit.
type Limits struct {
	// MaxProgramBytes bounds the source text (protocol MaxProgramBytes).
	MaxProgramBytes int
	// MaxTokens bounds the lexer output per parse.
	MaxTokens int
	// MaxStatements bounds the number of statements per program.
	MaxStatements int
	// MaxDepth bounds expression nesting, materialization and evaluation depth.
	MaxDepth int
	// MaxASTNodes bounds expression nodes per parse.
	MaxASTNodes int
	// MaxMaterializedNodes bounds parse-time lowering steps.
	MaxMaterializedNodes int
	// MaxEvalSteps bounds evaluation steps per render or action step.
	MaxEvalSteps int
	// MaxRenderNodes bounds component instances in one rendered tree.
	MaxRenderNodes int
	// MaxListItems bounds arrays produced by built-ins, functions and @Each.
	MaxListItems int
	// MaxStateKeys bounds the keys of the state store.
	MaxStateKeys int
	// MaxStateBytes bounds the persisted state document.
	MaxStateBytes int
	// MaxValueNodes bounds the nodes of a value returned by a function.
	MaxValueNodes int
}

// DefaultLimits are the limits used when a field is zero.
var DefaultLimits = Limits{
	MaxProgramBytes:      256 << 10,
	MaxTokens:            40000,
	MaxStatements:        4000,
	MaxDepth:             80,
	MaxASTNodes:          20000,
	MaxMaterializedNodes: 30000,
	MaxEvalSteps:         50000,
	MaxRenderNodes:       12000,
	MaxListItems:         10000,
	MaxStateKeys:         1000,
	MaxStateBytes:        48 << 10,
	MaxValueNodes:        30000,
}

func (l Limits) withDefaults() Limits {
	d := DefaultLimits
	set := func(dst *int, v int) {
		if v != 0 {
			*dst = v
		}
	}
	set(&d.MaxProgramBytes, l.MaxProgramBytes)
	set(&d.MaxTokens, l.MaxTokens)
	set(&d.MaxStatements, l.MaxStatements)
	set(&d.MaxDepth, l.MaxDepth)
	set(&d.MaxASTNodes, l.MaxASTNodes)
	set(&d.MaxMaterializedNodes, l.MaxMaterializedNodes)
	set(&d.MaxEvalSteps, l.MaxEvalSteps)
	set(&d.MaxRenderNodes, l.MaxRenderNodes)
	set(&d.MaxListItems, l.MaxListItems)
	set(&d.MaxStateKeys, l.MaxStateKeys)
	set(&d.MaxStateBytes, l.MaxStateBytes)
	set(&d.MaxValueNodes, l.MaxValueNodes)
	return d
}

// Diagnostic is an error or warning in the shape of spec section 8.3.
type Diagnostic struct {
	Source      string `json:"source"`
	Code        string `json:"code"`
	Message     string `json:"message"`
	StatementID string `json:"statementId,omitempty"`
	Component   string `json:"component,omitempty"`
	Path        string `json:"path,omitempty"`
	Hint        string `json:"hint,omitempty"`
	ToolName    string `json:"toolName,omitempty"`
	Severity    string `json:"severity,omitempty"`
}

// Warning reports whether the page renders through the diagnostic.
func (d Diagnostic) Warning() bool { return d.Severity == "warning" }

// Diagnostic sources and codes beyond the spec table.
const (
	SourceParser   = "parser"
	SourceRuntime  = "runtime"
	SourceQuery    = "query"
	SourceMutation = "mutation"
	SourceAction   = "action"

	CodeBudgetExceeded     = "budget-exceeded"
	CodeParseException     = "parse-exception"
	CodeRuntimeError       = "runtime-error"
	CodeInvalidStateDflt   = "invalid-state-default"
	CodeIncompleteInput    = "incomplete-input"
	CodeUnresolvedRef      = "unresolved-reference"
	CodeDataUnavailable    = "data-unavailable"
	CodeProgramTooLarge    = "program-too-large"
	CodeSchemaInvalid      = "schema-invalid"
	CodeActionInvalid      = "action-invalid"
	CodeStaleRevision      = "stale-revision"
	CodeStateTooLarge      = "state-too-large"
	CodeUnsupportedProgram = "unsupported-language"
)

// ── Source preparation (spec section 1.2, upstream parser.ts) ──

func skipDQString(s string, start int) int {
	if s[start] != '"' {
		return start
	}
	i := start + 1
	for i < len(s) {
		switch s[i] {
		case '\\':
			i += 2
		case '"':
			return i + 1
		default:
			i++
		}
	}
	return len(s)
}

func isFenceAt(s string, i int) bool {
	return i+2 < len(s) && s[i] == '`' && s[i+1] == '`' && s[i+2] == '`'
}

// StripFences extracts the program from markdown code fences, or returns the
// text unchanged when it has none. Backtick runs inside double-quoted
// strings do not open or close a fence. An unterminated fence runs to the end.
func StripFences(input string) string {
	if !strings.Contains(input, "```") {
		return input
	}
	var blocks []string
	i := 0
	for i < len(input) {
		fenceStart := -1
		for i < len(input) {
			if next := skipDQString(input, i); next > i {
				i = next
				continue
			}
			if isFenceAt(input, i) {
				fenceStart = i
				break
			}
			i++
		}
		if fenceStart == -1 {
			break
		}
		j := fenceStart + 3
		for j < len(input) && input[j] != '\n' {
			j++
		}
		if j >= len(input) {
			// Opening fence without a newline yet (streaming): drop the tag line.
			rest := input[fenceStart+3:]
			if k := strings.IndexByte(rest, '\n'); k >= 0 {
				rest = rest[k+1:]
			} else {
				rest = ""
			}
			blocks = append(blocks, rest)
			break
		}
		j++
		closePos := -1
		k := j
		for k < len(input) {
			if next := skipDQString(input, k); next > k {
				k = next
				continue
			}
			if isFenceAt(input, k) {
				closePos = k
				break
			}
			k++
		}
		if closePos != -1 {
			blocks = append(blocks, input[j:closePos])
			i = closePos + 3
		} else {
			blocks = append(blocks, input[j:])
			i = len(input)
		}
	}
	if len(blocks) > 0 {
		return strings.Join(blocks, "\n")
	}
	if strings.HasPrefix(input, "```") {
		j := 3
		for j < len(input) && input[j] != '\n' {
			j++
		}
		start := 3
		if j < len(input) {
			start = j + 1
		}
		body := input[start:]
		if t := strings.LastIndex(body, "```"); t != -1 {
			return body[:t]
		}
		return body
	}
	return input
}

// stripComments removes // and # line comments outside strings. A string
// may span lines; its state carries across lines.
func stripComments(input string) string {
	if !strings.ContainsAny(input, "/#") {
		return input
	}
	var inStr byte
	lines := strings.Split(input, "\n")
	for li, line := range lines {
		for i := 0; i < len(line); i++ {
			c := line[i]
			if inStr != 0 {
				if c == '\\' && i+1 < len(line) {
					i++
					continue
				}
				if c == inStr {
					inStr = 0
				}
				continue
			}
			if c == '"' || c == '\'' {
				inStr = c
				continue
			}
			if c == '/' && i+1 < len(line) && line[i+1] == '/' || c == '#' {
				lines[li] = strings.TrimRight(line[:i], " \t\r\n\v\f")
				break
			}
		}
	}
	return strings.Join(lines, "\n")
}

// Preprocess applies fence extraction, comment stripping and trimming.
func Preprocess(input string) string {
	return trimJSSpace(stripComments(StripFences(trimJSSpace(input))))
}

// autoClose closes an unterminated string and unclosed brackets.
func autoClose(input string) (string, bool) {
	var stack []byte
	var inStr byte
	esc := false
	for i := 0; i < len(input); i++ {
		c := input[i]
		if esc {
			esc = false
			continue
		}
		if c == '\\' && inStr != 0 {
			esc = true
			continue
		}
		if inStr != 0 {
			if c == inStr {
				inStr = 0
			}
			continue
		}
		switch c {
		case '"', '\'':
			inStr = c
		case '(', '[', '{':
			stack = append(stack, c)
		case ')':
			if len(stack) > 0 && stack[len(stack)-1] == '(' {
				stack = stack[:len(stack)-1]
			}
		case ']':
			if len(stack) > 0 && stack[len(stack)-1] == '[' {
				stack = stack[:len(stack)-1]
			}
		case '}':
			if len(stack) > 0 && stack[len(stack)-1] == '{' {
				stack = stack[:len(stack)-1]
			}
		}
	}
	if inStr == 0 && len(stack) == 0 {
		return input, false
	}
	var b strings.Builder
	b.Grow(len(input) + len(stack) + 2)
	b.WriteString(input)
	if inStr != 0 {
		if esc {
			b.WriteByte('\\')
		}
		b.WriteByte(inStr)
	}
	for j := len(stack) - 1; j >= 0; j-- {
		switch stack[j] {
		case '(':
			b.WriteByte(')')
		case '[':
			b.WriteByte(']')
		default:
			b.WriteByte('}')
		}
	}
	return b.String(), true
}

// rawStmt is one `name = expression` statement before expression parsing.
type rawStmt struct {
	id    string
	state bool // the name is a $variable
	toks  []token
}

// splitStatements splits tokens into statements at depth-zero newlines,
// keeping multi-line ternaries together (spec section 1.4).
func splitStatements(toks []token) []rawStmt {
	var out []rawStmt
	pos := 0
	n := len(toks)
	skipLine := func() {
		for pos < n && toks[pos].k != tNewline && toks[pos].k != tEOF {
			pos++
		}
	}
	for pos < n {
		for pos < n && toks[pos].k == tNewline {
			pos++
		}
		if pos >= n || toks[pos].k == tEOF {
			break
		}
		tok := toks[pos]
		if tok.k != tIdent && tok.k != tType && tok.k != tStateVar {
			skipLine()
			continue
		}
		pos++
		if pos >= n || toks[pos].k != tEquals {
			skipLine()
			continue
		}
		pos++
		start := pos
		var expr []token
		depth, ternary := 0, 0
		contiguous := true
		breakRun := func() {
			if contiguous {
				expr = append(expr, toks[start:pos]...)
				contiguous = false
			}
		}
		for pos < n && toks[pos].k != tEOF {
			k := toks[pos].k
			if k == tNewline && depth <= 0 && ternary <= 0 {
				peek := pos + 1
				for peek < n && toks[peek].k == tNewline {
					peek++
				}
				next := tEOF
				if peek < n {
					next = toks[peek].k
				}
				if next == tQuestion || next == tColon && ternary > 0 {
					breakRun()
					pos++
					continue
				}
				break
			}
			if k == tNewline {
				breakRun()
				pos++
				continue
			}
			switch {
			case k == tLParen || k == tLBrack || k == tLBrace:
				depth++
			case (k == tRParen || k == tRBrack || k == tRBrace) && depth > 0:
				depth--
			case k == tQuestion && depth == 0:
				ternary++
			case k == tColon && depth == 0 && ternary > 0:
				ternary--
			}
			if !contiguous {
				expr = append(expr, toks[pos])
			}
			pos++
		}
		if contiguous {
			expr = toks[start:pos]
		}
		if len(expr) > 0 {
			out = append(out, rawStmt{id: tok.s, state: tok.k == tStateVar, toks: expr})
		}
	}
	return out
}
