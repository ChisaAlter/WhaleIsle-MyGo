package market

import (
	"net/url"
	"regexp"
	"strings"
)

var npmName = regexp.MustCompile(`^(?:@[A-Za-z0-9][A-Za-z0-9._-]{0,63}/)?[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
var ghOwner = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,38}$`)
var ghRepo = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,99}$`)
var ghRef = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]{0,254}$`)
var safePath = regexp.MustCompile(`^[A-Za-z0-9_./-]+$`)

func validNPM(s string) bool {
	return len(s) <= 214 && !strings.Contains(s, "..") && npmName.MatchString(s)
}
func validRepo(owner, repo string) bool {
	return ghOwner.MatchString(owner) && !strings.HasSuffix(owner, "-") && ghRepo.MatchString(repo) && !strings.HasSuffix(repo, ".")
}
func sourceRepo(raw string) (owner, repo, subpath string) {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return
	}
	owner, repo = parts[0], strings.TrimSuffix(parts[1], ".git")
	if !validRepo(owner, repo) {
		return "", "", ""
	}
	if len(parts) > 2 {
		if len(parts) < 5 || parts[2] != "tree" || !ghRef.MatchString(parts[3]) {
			return "", "", ""
		}
		subpath = strings.Join(parts[4:], "/")
		if !validPath(subpath) {
			return "", "", ""
		}
	}
	return
}
func validPath(s string) bool {
	if s == "" || len(s) > 512 || !safePath.MatchString(s) || strings.Contains(s, "..") || strings.HasPrefix(s, "/") || strings.HasSuffix(s, "/") {
		return false
	}
	for _, p := range strings.Split(s, "/") {
		if p == "" || p == "." {
			return false
		}
	}
	return true
}
func parseGitSpec(spec string) (owner, repo, tail string) {
	if !strings.HasPrefix(spec, "github:") {
		return
	}
	p := strings.SplitN(strings.TrimPrefix(spec, "github:"), "#", 2)
	r := strings.Split(p[0], "/")
	if len(r) != 2 || !validRepo(r[0], r[1]) {
		return
	}
	owner, repo = r[0], r[1]
	if len(p) == 2 {
		tail = p[1]
		if strings.HasPrefix(tail, "path:/") {
			if !validPath(strings.TrimPrefix(tail, "path:/")) {
				return "", "", ""
			}
		} else if !ghRef.MatchString(tail) || strings.Contains(tail, "..") || strings.Contains(tail, "@{") || strings.HasSuffix(tail, ".") || strings.HasSuffix(tail, "/") {
			return "", "", ""
		}
	}
	return
}

// GitHubIdentity retains a monorepo subpath, but ignores a revision.
func GitHubIdentity(spec string) string {
	s := strings.TrimSpace(spec)
	if strings.HasPrefix(s, "git+https://github.com/") {
		s = "github:" + strings.TrimPrefix(s, "git+https://github.com/")
	}
	if strings.HasPrefix(s, "https://github.com/") {
		s = "github:" + strings.TrimPrefix(s, "https://github.com/")
	}
	p := strings.SplitN(s, "#", 2)
	p[0] = strings.TrimSuffix(p[0], ".git")
	s = strings.Join(p, "#")
	o, r, t := parseGitSpec(s)
	if o == "" {
		return ""
	}
	id := strings.ToLower(o + "/" + r)
	if strings.HasPrefix(t, "path:/") {
		id += "#" + strings.ToLower(t)
	}
	return id
}
func dropped(s string) bool {
	if i := strings.LastIndex(s, "/"); i >= 0 {
		s = s[i+1:]
	}
	for _, x := range []string{"dsh-genui", "dsh-plugin-yet-another-subagent", "dshmarket", "dsh-im", "xmanrui-dsh-im", "dsh-usage-panel", "dshbot", "dsh-whale", "dsh-remote"} {
		if s == x {
			return true
		}
	}
	return false
}
func ValidInstallItem(i CatalogItem) bool {
	if i.Deprecated || i.ID == "" || dropped(i.PackageName) || dropped(i.Repo) || i.InstallSpec == "" {
		return false
	}
	if validNPM(i.InstallSpec) {
		return i.InstallSpec == i.PackageName
	}
	o, r, _ := parseGitSpec(i.InstallSpec)
	ho, hr, _ := sourceRepo(i.Homepage)
	return o != "" && o == ho && r == hr && !dropped(r)
}
func resolveSpec(npm, home, command string) string {
	if validNPM(npm) {
		return npm
	}
	o, r, p := sourceRepo(home)
	if o != "" {
		s := "github:" + o + "/" + r
		if p != "" {
			s += "#path:/" + p
		}
		return s
	}
	tokens := strings.Fields(command)
	if len(tokens) == 0 {
		return ""
	}
	s := strings.Trim(tokens[len(tokens)-1], `"'`)
	if ValidInstallItem(CatalogItem{ID: "candidate", PackageName: npm, Homepage: home, InstallSpec: s}) {
		return s
	}
	return ""
}
