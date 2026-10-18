package installctrl

import (
	"regexp"
	"strings"
)

const maxAllowBuilds = 32

var (
	githubOwnerPattern = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,38})$`)
	githubRepoPattern  = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9._-]{0,99})$`)
	githubRefPattern   = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9._/-]{0,254})$`)
	packageAllowBuild  = regexp.MustCompile(`^(?:@[A-Za-z0-9][A-Za-z0-9._-]{0,63}/)?[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
	githubAllowBuild   = regexp.MustCompile(`^github\.com/[A-Za-z0-9][A-Za-z0-9-]{0,38}/[A-Za-z0-9][A-Za-z0-9._-]{0,99}$`)
	gitAllowBuild      = regexp.MustCompile(`^[A-Za-z0-9@/_.-]+@git\+https://github\.com/[A-Za-z0-9._-]+/[A-Za-z0-9._-]+\.git$`)
	githubSpecPattern  = regexp.MustCompile(`^github:([^/#]+)/([^/#]+)(?:#([^#]+))?$`)
)

// IsValidGithubSpec reports whether spec is github:owner/repo with optional #ref.
func IsValidGithubSpec(spec string) bool {
	value := strings.TrimSpace(spec)
	m := githubSpecPattern.FindStringSubmatch(value)
	if m == nil {
		return false
	}
	owner, repo, ref := m[1], m[2], m[3]
	if !githubOwnerPattern.MatchString(owner) || !githubRepoPattern.MatchString(repo) {
		return false
	}
	if strings.HasSuffix(owner, "-") || repo == "." || repo == ".." || strings.HasSuffix(repo, ".") {
		return false
	}
	if ref == "" {
		return true
	}
	return githubRefPattern.MatchString(ref) &&
		!strings.Contains(ref, "..") &&
		!strings.Contains(ref, "@{") &&
		!strings.HasSuffix(ref, ".") &&
		!strings.HasSuffix(ref, "/")
}

func isValidPackageName(value string) bool {
	name := strings.TrimSpace(value)
	if name == "" || len(name) > 214 || name == "." || name == ".." || strings.Contains(name, "..") {
		return false
	}
	return packageAllowBuild.MatchString(name)
}

func isValidAllowBuild(value string) bool {
	key := strings.TrimSpace(value)
	if key == "" || len(key) > 214 || key == "." || key == ".." || strings.Contains(key, "..") {
		return false
	}
	return isValidPackageName(key) || githubAllowBuild.MatchString(key) || gitAllowBuild.MatchString(key)
}

// NormalizeAllowBuilds deduplicates valid allowBuilds keys, or returns nil when invalid.
func NormalizeAllowBuilds(value []string) []string {
	if value == nil {
		return []string{}
	}
	if len(value) > maxAllowBuilds {
		return nil
	}
	keys := make([]string, 0, len(value))
	seen := make(map[string]struct{}, len(value))
	for _, item := range value {
		if !isValidAllowBuild(item) {
			return nil
		}
		key := strings.TrimSpace(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	return keys
}

var registryVersionSpec = regexp.MustCompile(`^[~^]?[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)

// RegistryPluginName accepts only name@exact, name@^version or name@~version.
// Dist-tags, aliases, arbitrary ranges and URL/local specs are intentionally rejected.
func RegistryPluginName(spec string) string {
	value := strings.TrimSpace(spec)
	at := strings.LastIndexByte(value, '@')
	if at <= 0 {
		return ""
	}
	name, version := value[:at], value[at+1:]
	if name != strings.TrimSpace(name) || !isValidPackageName(name) || !registryVersionSpec.MatchString(version) {
		return ""
	}
	return name
}
