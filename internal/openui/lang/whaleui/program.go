package whaleui

import (
	"strings"

	"whaleisle/internal/openui/lang"
)

// Prepared is a program ready for the OpenUI Lang runtime.
type Prepared struct {
	Program     string
	Diagnostics []lang.Diagnostic
	// Panel is set when a dsh-ui spec asks to open as a panel.
	Panel, Append bool
}

// PrepareProgram converts the wire program of a language tag into OpenUI
// Lang. A dsh-ui spec (or a whale-ui program that is a JSON object) is
// wrapped as `root = Col([genui])` with `genui = Genui(spec)`, as the viewer
// does. While streaming, a partial spec is closed best-effort.
func PrepareProgram(language, program string, streaming bool) Prepared {
	isSpec := language == "dsh-ui" || (language == "whale-ui" || language == "") && strings.HasPrefix(strings.TrimLeft(program, " \t\r\n"), "{")
	if !isSpec {
		if language != "" && !ValidLanguage(language) {
			return Prepared{Program: "root = Col([])", Diagnostics: []lang.Diagnostic{{Source: lang.SourceParser, Code: lang.CodeUnsupportedProgram, Message: "The interface language is unsupported."}}}
		}
		return Prepared{Program: program}
	}
	spec, complete := parseSpec(program, streaming)
	o, ok := spec.(*lang.Object)
	if !ok {
		p := Prepared{Program: "root = Col([])"}
		if !streaming {
			p.Diagnostics = []lang.Diagnostic{{Source: lang.SourceParser, Code: "genui-invalid", Message: "The declarative interface is incomplete."}}
		}
		return p
	}
	out := o.Clone()
	panel := out.Value("panel") == true
	appendFlag := out.Value("append") == true
	out.Delete("panel")
	out.Delete("append")
	p := Prepared{Program: "root = Col([genui])\ngenui = Genui(" + string(lang.EncodeJSON(out)) + ")", Panel: panel, Append: appendFlag && panel}
	if !complete && !streaming {
		p.Diagnostics = []lang.Diagnostic{{Source: lang.SourceParser, Code: "genui-invalid", Message: "The declarative interface is incomplete."}}
	}
	return p
}

// parseSpec parses a JSON spec; while streaming it closes an unterminated
// tail (open string, trailing separator, open brackets).
func parseSpec(text string, streaming bool) (lang.Value, bool) {
	text = strings.TrimSpace(text)
	if v, err := lang.ParseJSON([]byte(text)); err == nil {
		return v, true
	}
	if !streaming {
		return nil, false
	}
	for cut := len(text); cut > 0; {
		if v, err := lang.ParseJSON([]byte(closeJSON(text[:cut]))); err == nil {
			return v, false
		}
		// Drop back to the previous structural boundary and try again.
		idx := strings.LastIndexAny(text[:cut-1], ",{[")
		if idx < 0 {
			break
		}
		cut = idx + 1
		if text[idx] == ',' {
			cut = idx
		}
	}
	return nil, false
}

func closeJSON(s string) string {
	var stack []byte
	inStr, esc := false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr {
			switch {
			case esc:
				esc = false
			case c == '\\':
				esc = true
			case c == '"':
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case '{', '[':
			stack = append(stack, c)
		case '}', ']':
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	out := s
	if inStr {
		if esc {
			out = out[:len(out)-1]
		}
		out += `"`
	}
	out = strings.TrimRight(out, " \t\r\n")
	out = strings.TrimSuffix(out, ",")
	if strings.HasSuffix(out, ":") {
		out += "null"
	}
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i] == '{' {
			out += "}"
		} else {
			out += "]"
		}
	}
	return out
}
