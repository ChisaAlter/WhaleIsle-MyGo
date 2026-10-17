package installctrl

import (
	"regexp"
	"strings"
)

var (
	ignoredBuildScriptsRE = regexp.MustCompile(`(?i)ignored build scripts:\s*([^\n]+)`)
	gitHostedPrepareRE    = regexp.MustCompile(`git-hosted package "([^"]+)" needs to execute build scripts`)
	allowBuildsKeyRE        = regexp.MustCompile(`["'](@?[\w.-]+(?:[/.][\w.-]+)*)["']\s*:\s*(?:true|false)`)
	indentedNameRE          = regexp.MustCompile(`(?m)^\s{2,}([@a-z0-9._/-]+(?:\/[a-z0-9._-]+)?)\s*$`)
)

// ParseAllowBuilds extracts pnpm allow-build package keys from install log output.
// Mirrors Electron marketplace-allowbuilds.js.
func ParseAllowBuilds(log string) []string {
	text := strings.ReplaceAll(log, `\"`, `"`)
	names := make(map[string]struct{})

	if m := ignoredBuildScriptsRE.FindStringSubmatch(text); len(m) > 1 {
		for _, part := range strings.FieldsFunc(m[1], func(r rune) bool {
			return r == ',' || r == ' ' || r == '\t'
		}) {
			name := strings.TrimSpace(part)
			if at := strings.Index(name, "@"); at > 0 && len(name) > at+1 && name[at+1] >= '0' && name[at+1] <= '9' {
				name = name[:at]
			}
			if name != "" && !strings.HasPrefix(strings.ToLower(name), "http") {
				if isValidAllowBuild(name) {
					names[name] = struct{}{}
				}
			}
		}
	}

	if m := gitHostedPrepareRE.FindStringSubmatch(text); len(m) > 1 {
		raw := strings.TrimSpace(m[1])
		name := raw
		if at := strings.LastIndex(raw, "@"); at > 0 {
			name = raw[:at]
		}
		if name != "" && isValidAllowBuild(name) {
			names[name] = struct{}{}
		}
	}

	for _, m := range indentedNameRE.FindAllStringSubmatch(text, -1) {
		name := m[1]
		switch strings.ToLower(name) {
		case "run", "add", "the", "following", "dependencies", "ignored":
			continue
		}
		if isValidAllowBuild(name) {
			names[name] = struct{}{}
		}
	}

	for _, m := range allowBuildsKeyRE.FindAllStringSubmatch(text, -1) {
		if isValidAllowBuild(m[1]) {
			names[m[1]] = struct{}{}
		}
	}

	out := make([]string, 0, len(names))
	for k := range names {
		out = append(out, k)
	}
	return out
}

// IsBuildApprovalFailure reports whether a non-zero exit with parsed allow-build
// keys indicates pnpm blocked prepare scripts (not a generic network failure).
func IsBuildApprovalFailure(exitCode int, allowBuilds []string) bool {
	return exitCode != 0 && len(allowBuilds) > 0
}
