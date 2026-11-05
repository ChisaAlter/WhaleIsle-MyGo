package market

import (
	"regexp"
	"strings"
)

var historyRedactions = []struct {
	pattern     *regexp.Regexp
	replacement string
}{
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
