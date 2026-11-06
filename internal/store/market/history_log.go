package market

import (
	"regexp"
	"strings"
)

var historyRedactions = []struct {
	pattern     *regexp.Regexp
	replacement string
}{
	// Authorization values can contain a scheme and several credentials (for
	// example Basic or Digest). Redact the complete field, not its first word.
	{regexp.MustCompile(`(?i)((?:(?:proxy-)?authorization|password|githubToken|api[_-]?key|_authToken|access_token)["']?[ \t]*[:=][ \t]*)"(?:\\.|[^"\\\r\n])*"`), `${1}"[redacted]"`},
	{regexp.MustCompile(`(?i)((?:(?:proxy-)?authorization|password|githubToken|api[_-]?key|_authToken|access_token)["']?[ \t]*[:=][ \t]*)'(?:\\.|[^'\\\r\n])*'`), `${1}'[redacted]'`},
	{regexp.MustCompile(`(?im)((?:proxy-)?authorization["']?[ \t]*[:=][ \t]*)[^"' \t\r\n][^\r\n]*`), `${1}[redacted]`},
	{regexp.MustCompile(`(?i)(https?://)[^\s/@]+(?::[^\s/@]*)?@`), `${1}[redacted]@`},
	{regexp.MustCompile(`(?i)([?&](?:token|key|api_key|access_token|auth|password)=)[^\s&#]+`), `${1}[redacted]`},
	{regexp.MustCompile(`(?i)((?:authorization|password|githubToken|api[_-]?key|_authToken|access_token)["']?\s*[:=]\s*["']?)(?:Bearer\s+)?[^\s,"'}]+`), `${1}[redacted]`},
	{regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9_]+|github_pat_[A-Za-z0-9_]+|sk-[A-Za-z0-9_-]+)\b`), `[redacted]`},
}

// RedactOperationLog is shared by durable history and export. Keep the trailing
// bounded diagnostic, as the installed market does, without splitting UTF-8.
func RedactOperationLog(value string, limit int) string {
	value = strings.ReplaceAll(value, "\x00", "")
	for _, rule := range historyRedactions {
		value = rule.pattern.ReplaceAllString(value, rule.replacement)
	}
	runes := []rune(value)
	if len(runes) > limit {
		value = string(runes[len(runes)-limit:])
	}
	return value
}
