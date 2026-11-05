package market

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var githubCommit = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)
var pinnedGithubCommit = regexp.MustCompile(`#(?:commit=)?([0-9a-fA-F]{7,40})(?:&|$)`)
var lockedGithubCommit = regexp.MustCompile(`codeload\.github\.com/([^/\s]+/[^/\s]+)/tar\.gz/([0-9a-fA-F]{40})`)

// InstalledGitCommit uses only the confirmed profile's lockfile or the installed
// dependency's pinned commit. A repo with multiple lock commits is ambiguous.
func InstalledGitCommit(item CatalogItem, installedSpec, profileDir string) string {
	identity := GitHubIdentity(item.InstallSpec)
	if identity == "" || GitHubIdentity(installedSpec) != identity {
		return ""
	}
	repo := strings.SplitN(identity, "#", 2)[0]
	if profileDir != "" {
		if raw, err := os.ReadFile(filepath.Join(profileDir, "pnpm-lock.yaml")); err == nil {
			found := ""
			for _, m := range lockedGithubCommit.FindAllStringSubmatch(string(raw), -1) {
				if strings.ToLower(m[1]) != repo {
					continue
				}
				next := strings.ToLower(m[2])
				if found != "" && next != found {
					return ""
				}
				found = next
			}
			if found != "" {
				return found
			}
		}
	}
	if m := pinnedGithubCommit.FindStringSubmatch(installedSpec); m != nil {
		return strings.ToLower(m[1])
	}
	return ""
}

// LatestCommit resolves HEAD through the same public GitHub source as the
// installed desktop market. No Host cookie or account credential is forwarded.
func (s *Service) LatestCommit(ctx context.Context, item CatalogItem) (string, error) {
	s.mu.Lock()
	known, ok := s.known[item.ID]
	s.mu.Unlock()
	if !ok || !sameCatalogItem(known, item) || !ValidInstallItem(known) || GitHubIdentity(known.InstallSpec) == "" {
		return "", errors.New("No verified GitHub update source")
	}
	owner, repo, _ := parseGitSpec(known.InstallSpec)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	raw, err := publicBody(ctx, s.client, "https://api.github.com/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(repo)+"/commits/HEAD", "application/vnd.github+json", 256*1024)
	if err != nil {
		return "", err
	}
	var result struct {
		SHA string `json:"sha"`
	}
	if json.Unmarshal(raw, &result) != nil || !githubCommit.MatchString(result.SHA) {
		return "", errors.New("Invalid GitHub commit metadata")
	}
	return strings.ToLower(result.SHA), nil
}

func GitHubUpdateSpec(item CatalogItem, commit string) string {
	if !githubCommit.MatchString(commit) {
		return ""
	}
	owner, repo, tail := parseGitSpec(item.InstallSpec)
	if owner == "" {
		return ""
	}
	// The Host's path installer owns resolving/packing monorepo targets.
	if strings.HasPrefix(tail, "path:/") {
		return item.InstallSpec
	}
	return "github:" + owner + "/" + repo + "#" + strings.ToLower(commit)
}
