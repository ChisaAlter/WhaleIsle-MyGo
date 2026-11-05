package chat

import (
	"regexp"
	"sort"
	"strings"
)

type secretRange struct {
	start, end int
	kind       string
}
type credentialPattern struct {
	re    *regexp.Regexp
	kind  string
	digit bool
}

var credentialPatterns = []credentialPattern{
	{regexp.MustCompile(`sk-[A-Za-z0-9_-]{20,}`), "prefix", true},
	{regexp.MustCompile(`AIza[A-Za-z0-9_-]{35}`), "prefix", false},
	{regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{36,}|github_pat_[A-Za-z0-9_]{22,}|glpat-[A-Za-z0-9_-]{20,}|npm_[A-Za-z0-9]{36,}|hf_[A-Za-z0-9]{30,}`), "prefix", false},
	{regexp.MustCompile(`(?:AKIA|ASIA)[A-Z0-9]{16}\b|LTAI[A-Za-z0-9]{12,20}\b|AKID[A-Za-z0-9]{28,36}\b`), "prefix", false},
	{regexp.MustCompile(`xox[abprs]-[A-Za-z0-9-]{10,}|[sr]k_(?:live|test)_[A-Za-z0-9]{16,}|whsec_[A-Za-z0-9]{24,}`), "prefix", false},
	{regexp.MustCompile(`eyJ[A-Za-z0-9_-]{8,}\.eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}`), "jwt", false},
}
var credentialAddress = regexp.MustCompile(`\b[a-zA-Z][a-zA-Z0-9+.-]{1,15}://[^\s/:@?#]{0,256}:([^\s/@?#]+)@`)
var credentialPrivate = regexp.MustCompile(`-----BEGIN [A-Z0-9 ]{0,40}PRIVATE KEY-----`)
var credentialPrivateEnd = regexp.MustCompile(`-----END [A-Z0-9 ]{0,40}PRIVATE KEY-----`)

func userSecrets(text string) []secretRange {
	var ranges []secretRange
	for _, loc := range credentialPrivate.FindAllStringIndex(text, -1) {
		end := len(text)
		if tail := credentialPrivateEnd.FindStringIndex(text[loc[1]:]); tail != nil {
			end = loc[1] + tail[0]
		}
		if loc[1] < end {
			ranges = append(ranges, secretRange{loc[1], end, "private"})
		}
	}
	for _, loc := range credentialAddress.FindAllStringSubmatchIndex(text, -1) {
		ranges = append(ranges, secretRange{loc[2], loc[3], "password"})
	}
	for _, p := range credentialPatterns {
		for _, loc := range p.re.FindAllStringIndex(text, -1) {
			if loc[0] > 0 {
				b := text[loc[0]-1]
				if b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' || b >= '0' && b <= '9' || b == '_' {
					continue
				}
			}
			if p.digit && !strings.ContainsAny(text[loc[0]+3:min(loc[1], loc[0]+259)], "0123456789") {
				continue
			}
			if p.kind == "jwt" {
				first, _, _ := strings.Cut(text[loc[0]:loc[1]], ".")
				if len(first)-3 > 2048 {
					continue
				}
			}
			ranges = append(ranges, secretRange{loc[0], loc[1], p.kind})
		}
	}
	sort.SliceStable(ranges, func(i, j int) bool { return ranges[i].start < ranges[j].start })
	merged := ranges[:0]
	for _, r := range ranges {
		if len(merged) > 0 && r.start < merged[len(merged)-1].end {
			merged[len(merged)-1].end = max(merged[len(merged)-1].end, r.end)
		} else {
			merged = append(merged, r)
		}
	}
	return merged
}
func maskedCredential(raw, kind string) string {
	var b strings.Builder
	for i, r := range raw {
		if r == '\r' || r == '\n' || (kind == "prefix" || kind == "jwt") && i < 8 {
			b.WriteRune(r)
		} else {
			b.WriteByte('*')
		}
	}
	return b.String()
}
func maskedUserText(text string) string {
	var out strings.Builder
	start := 0
	for _, r := range userSecrets(text) {
		out.WriteString(text[start:r.start])
		out.WriteString(maskedCredential(text[r.start:r.end], r.kind))
		start = r.end
	}
	out.WriteString(text[start:])
	return out.String()
}
func visibleUserReferenceRuns(text string, facts userReferenceFacts) []userReferenceRun {
	secrets := userSecrets(text)
	if len(secrets) == 0 {
		return userReferenceRuns(text, facts)
	}
	var out []userReferenceRun
	offset := 0
	for _, run := range userReferenceRuns(text, facts) {
		raw := run.Raw
		if raw == "" {
			raw = run.Text
		}
		end := offset + len(raw)
		start := offset
		touched := false
		for _, r := range secrets {
			if r.end <= offset || r.start >= end {
				continue
			}
			touched = true
			a, b := max(start, r.start), min(end, r.end)
			if start < a {
				out = append(out, userReferenceRun{Text: text[start:a]})
			}
			out = append(out, userReferenceRun{Text: maskedCredential(text[a:b], r.kind), Raw: text[a:b], Kind: "secret"})
			start = b
		}
		if !touched {
			out = append(out, run)
		} else if start < end {
			out = append(out, userReferenceRun{Text: text[start:end]})
		}
		offset = end
	}
	return out
}
