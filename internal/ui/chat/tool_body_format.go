package chat

import "encoding/json"

// formatToolCodeProgram follows formatToolBody('code', argsRaw): only an empty
// raw payload is absent; invalid JSON stays literal; a nonempty string code
// property is returned exactly as authored. Other values use JSON.parse/stringify
// ordering, decoded strings, and IEEE-754 numbers through the existing formatter.
// This only derives display text; it never rewrites the retained arguments.
//
// Native decoding shares encoding/json's limits: lone UTF-16 surrogates become
// U+FFFD, and JSON nested beyond its depth limit is retained as invalid raw text.
func formatToolCodeProgram(arguments string) (string, bool) {
	if arguments == "" {
		return "", false
	}
	raw := json.RawMessage(arguments)
	if !json.Valid(raw) {
		return arguments, true
	}
	if fields, ok := contextObjectEntries(raw); ok {
		if code, ok := contextSourceString(fields, "code"); ok && code != "" {
			return code, true
		}
	}
	return contextFormattedJSON(raw), true
}
