package chat

import (
	"encoding/json"
	"strings"

	"whaleisle/internal/store/session"
)

// toolSearchModel is the installed settled grep/glob card's display data. Paths
// retain their recorded spelling and confer no filesystem or navigation authority.
// Total and LineNumber retain JavaScript Number.isInteger semantics, including
// integral IEEE-754 values outside the native int and safe-integer ranges.
type toolSearchModel struct {
	Kind      string
	Paths     []string
	Files     []toolSearchFile
	Total     float64
	Truncated bool
	Recovery  string
}

type toolSearchFile struct {
	Path    string
	Matches []toolSearchMatch
}

type toolSearchMatch struct {
	LineNumber float64
	Line       string
}

// toolSearchModelFor derives only a successful, settled root search with a
// retained call head and valid root receipt. Metadata owns the card rows; text
// blocks supply recovery only when truncated. Source calls and events stay intact.
func toolSearchModelFor(call *session.ToolCallView) *toolSearchModel {
	if call == nil || call.Name != "grep" && call.Name != "glob" || call.Call == nil || call.Result == nil || call.State != "ok" || call.ParentCallID != "" || call.RootCallID != "" && call.RootCallID != call.CallID {
		return nil
	}
	args, ok := terminalModelObject(json.RawMessage(call.Arguments))
	if !ok {
		return nil
	}
	pattern, ok := terminalModelString(args["pattern"])
	if !ok || call.Name == "grep" && pattern == "" || call.Name == "glob" && terminalModelBlank(pattern) {
		return nil
	}
	if raw, exists := args["path"]; exists {
		path, valid := terminalModelString(raw)
		if !valid || terminalModelBlank(path) {
			return nil
		}
	}
	if call.Name == "grep" {
		if raw, exists := args["include"]; exists {
			include, valid := terminalModelString(raw)
			if !valid || !toolSearchModelInclude(include) {
				return nil
			}
		}
	}
	// Reject malformed or nested wire receipts even if the lifecycle projection
	// looks like a successful root. A root receipt must have data.message object.
	envelope, ok := terminalModelObject(call.Result.Raw)
	if !ok {
		return nil
	}
	typ, ok := terminalModelString(envelope["type"])
	if !ok || typ != "tool/result" || call.Result.EventType != "" && call.Result.EventType != typ {
		return nil
	}
	data, ok := terminalModelObject(envelope["data"])
	if !ok {
		return nil
	}
	if _, ok := terminalModelObject(data["message"]); !ok {
		return nil
	}
	result := toolSummaryReadResult(call.Result)
	if result.nested || result.failed || result.stopped {
		return nil
	}
	meta, ok := terminalModelObject(result.meta)
	if !ok {
		return nil
	}
	truncated, ok := terminalModelBool(meta["truncated"])
	if !ok {
		return nil
	}
	total, ok := toolReadModelInteger(meta["total"], 0)
	if !ok {
		return nil
	}
	shape, ok := terminalModelString(meta["shape"])
	if !ok {
		return nil
	}
	model := &toolSearchModel{Kind: shape, Total: total, Truncated: truncated}
	if call.Name == "glob" {
		paths, valid := contextRawArray(meta["paths"])
		if shape != "paths" || !valid {
			return nil
		}
		model.Paths = make([]string, 0, len(paths))
		for _, raw := range paths {
			path, valid := terminalModelString(raw)
			if !valid {
				return nil
			}
			model.Paths = append(model.Paths, path)
		}
	} else {
		files, valid := contextRawArray(meta["files"])
		if shape != "matches" || !valid {
			return nil
		}
		model.Files = make([]toolSearchFile, 0, len(files))
		for _, raw := range files {
			file, valid := terminalModelObject(raw)
			if !valid {
				return nil
			}
			path, valid := terminalModelString(file["path"])
			if !valid {
				return nil
			}
			matches, valid := contextRawArray(file["matches"])
			if !valid {
				return nil
			}
			narrowed := toolSearchFile{Path: path, Matches: make([]toolSearchMatch, 0, len(matches))}
			for _, raw := range matches {
				match, valid := terminalModelObject(raw)
				if !valid {
					return nil
				}
				number, valid := toolReadModelInteger(match["lineNumber"], 1)
				if !valid {
					return nil
				}
				line, valid := terminalModelString(match["line"])
				if !valid {
					return nil
				}
				narrowed.Matches = append(narrowed.Matches, toolSearchMatch{LineNumber: number, Line: line})
			}
			model.Files = append(model.Files, narrowed)
		}
	}
	if truncated {
		content, _ := contextRawArray(result.content)
		parts := make([]string, 0, len(content))
		for _, raw := range content {
			block, _ := terminalModelObject(raw)
			if typ, _ := terminalModelString(block["type"]); typ == "text" {
				if text, valid := terminalModelString(block["text"]); valid {
					parts = append(parts, text)
				}
			}
		}
		model.Recovery = strings.Join(parts, "\n")
	}
	return model
}

// The original checks a literal leading ! and commas outside braces; it does
// not parse glob syntax, require balanced braces, trim before !, or honor escapes.
func toolSearchModelInclude(include string) bool {
	if terminalModelBlank(include) || strings.HasPrefix(include, "!") {
		return false
	}
	depth := 0
	for _, character := range include {
		switch character {
		case '{':
			depth++
		case '}':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				return false
			}
		}
	}
	return true
}
