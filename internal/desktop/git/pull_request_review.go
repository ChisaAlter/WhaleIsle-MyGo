package git

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// PullRequestPlan binds a draft PR's exact recipient, head/base and manual text.
// Private identity/token fields ensure edited text always requires a fresh review.
type PullRequestPlan struct {
	Repository, Base, Title, Body string
	Push                          PushPlan
	root, gitDir, head, token     string
}
type PullRequestResult struct {
	URL, Repository     string
	Existing, Uncertain bool
}

var githubRepositoryPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

func githubRepository(remote string) (string, error) {
	var path string
	if strings.HasPrefix(remote, "git@github.com:") {
		path = strings.TrimPrefix(remote, "git@github.com:")
	} else {
		u, e := url.Parse(remote)
		if e != nil || u.Host != "github.com" || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && u.Scheme != "ssh") {
			return "", errors.New("draft PR requires an unambiguous github.com push URL")
		}
		if u.User != nil && (u.Scheme != "ssh" || u.User.String() != "git") {
			return "", errors.New("credential-bearing GitHub URL is not supported")
		}
		path = strings.TrimPrefix(u.Path, "/")
	}
	path = strings.TrimSuffix(path, ".git")
	if !githubRepositoryPattern.MatchString(path) || strings.Contains(path, "..") {
		return "", errors.New("invalid GitHub repository in push URL")
	}
	return path, nil
}
func ValidPullRequestURL(raw, repository string) bool {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.RawPath != "" {
		return false
	}
	prefix := "/" + repository + "/pull/"
	if !strings.HasPrefix(u.Path, prefix) {
		return false
	}
	n, e := strconv.Atoi(strings.TrimPrefix(u.Path, prefix))
	return e == nil && n > 0
}
func workflowIdentity(ctx context.Context, root string) (string, string, error) {
	absolute, e := filepath.Abs(root)
	if e != nil {
		return "", "", e
	}
	absolute, e = filepath.EvalSymlinks(absolute)
	if e != nil {
		return "", "", e
	}
	dir, e := selectedText(ctx, absolute, "", "rev-parse", "--absolute-git-dir")
	return absolute, dir, e
}
func pullRequestToken(p PullRequestPlan) string {
	data, _ := json.Marshal([]any{p.Repository, p.Base, p.Title, p.Body, p.Push, p.root, p.gitDir, p.head})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
func PreparePullRequest(ctx context.Context, root string, push PushPlan, title, body string) (PullRequestPlan, error) {
	return preparePullRequest(ctx, root, push, title, body, combinedExecRunner)
}
func preparePullRequest(ctx context.Context, root string, push PushPlan, title, body string, run combinedRunner) (p PullRequestPlan, err error) {
	p.Push = push
	p.Title = strings.TrimSpace(title)
	p.Body = strings.TrimSpace(body)
	if p.Title == "" || strings.ContainsAny(p.Title, "\r\n") {
		return p, errors.New("enter a one-line PR title before review")
	}
	p.root, p.gitDir, err = workflowIdentity(ctx, root)
	if err != nil {
		return
	}
	p.head, err = selectedText(ctx, root, "", "rev-parse", "HEAD")
	if err != nil {
		return
	}
	current, e := PreparePush(root)
	if e != nil {
		return p, e
	}
	if current != push {
		return p, errors.New("push destination changed; review again")
	}
	p.Repository, err = githubRepository(push.URL)
	if err != nil {
		return
	}
	emitPhase(ctx, "Resolving PR base in "+p.Repository+"…")
	out, e := run(ctx, root, "gh", "repo", "view", p.Repository, "--json", "defaultBranchRef")
	if e != nil {
		return p, fmt.Errorf("cannot resolve PR base: %w: %s", e, strings.TrimSpace(string(out)))
	}
	var repo struct {
		DefaultBranchRef struct {
			Name string `json:"name"`
		} `json:"defaultBranchRef"`
	}
	if err = json.Unmarshal(out, &repo); err != nil {
		return
	}
	p.Base = repo.DefaultBranchRef.Name
	if p.Base == "" || checkBranchName(root, p.Base) != nil {
		return p, errors.New("GitHub did not return a valid default branch")
	}
	if p.Base == push.Target {
		return p, errors.New("create a feature branch before opening a PR against the default branch")
	}
	p.token = pullRequestToken(p)
	return
}
func validatePullRequest(ctx context.Context, root string, p PullRequestPlan, expectedHead string) error {
	if p.token == "" || p.token != pullRequestToken(p) {
		return errors.New("PR recipient or text changed; review again")
	}
	abs, dir, e := workflowIdentity(ctx, root)
	if e != nil {
		return e
	}
	if abs != p.root || dir != p.gitDir {
		return errors.New("PR repository changed; review again")
	}
	current, e := PreparePush(root)
	if e != nil {
		return e
	}
	if current != p.Push {
		return errors.New("PR branch or push destination changed; review again")
	}
	if expectedHead == "" {
		expectedHead = p.head
	}
	head, e := selectedText(ctx, root, "", "rev-parse", "HEAD")
	if e != nil {
		return e
	}
	if head != expectedHead {
		return errors.New("HEAD changed after review; review again")
	}
	return nil
}
func lookupPullRequest(ctx context.Context, root string, p PullRequestPlan, run combinedRunner) (PullRequestResult, error) {
	emitPhase(ctx, "Checking existing PR in "+p.Repository+"…")
	out, e := run(ctx, root, "gh", "pr", "list", "--repo", p.Repository, "--head", p.Push.Target, "--base", p.Base, "--state", "open", "--json", "number,title,url,state,headRefName,baseRefName,headRepository,headRepositoryOwner")
	if e != nil {
		return PullRequestResult{}, fmt.Errorf("cannot check existing PR; creation stopped: %w: %s", e, strings.TrimSpace(string(out)))
	}
	var list []struct {
		PRInfo
		HeadRefName, BaseRefName string
		HeadRepository           struct{ Name string }
		HeadRepositoryOwner      struct{ Login string }
	}
	e = json.Unmarshal(out, &list)
	if e != nil {
		return PullRequestResult{}, e
	}
	if len(list) > 1 {
		return PullRequestResult{}, errors.New("multiple open PRs match; inspect them on GitHub")
	}
	if len(list) == 0 {
		return PullRequestResult{}, nil
	}
	pinfo := list[0]
	if pinfo.State != "OPEN" || !ValidPullRequestURL(pinfo.URL, p.Repository) || pinfo.HeadRefName != p.Push.Target || pinfo.BaseRefName != p.Base || !strings.EqualFold(pinfo.HeadRepositoryOwner.Login+"/"+pinfo.HeadRepository.Name, p.Repository) {
		return PullRequestResult{}, errors.New("GitHub returned an unexpected PR URL or state")
	}
	return PullRequestResult{URL: pinfo.URL, Repository: p.Repository, Existing: true}, nil
}

// LookupPullRequestReviewed is read-only, including after an ambiguous create result.
func LookupPullRequestReviewed(ctx context.Context, root string, p PullRequestPlan, head string) (PullRequestResult, error) {
	if e := validatePullRequest(ctx, root, p, head); e != nil {
		return PullRequestResult{}, e
	}
	return lookupPullRequest(ctx, root, p, combinedExecRunner)
}
func CreatePullRequestReviewed(ctx context.Context, root string, p PullRequestPlan, head string) (PullRequestResult, error) {
	return createPullRequestReviewed(ctx, root, p, head, combinedExecRunner)
}
func createPullRequestReviewed(ctx context.Context, root string, p PullRequestPlan, head string, run combinedRunner) (PullRequestResult, error) {
	if e := validatePullRequest(ctx, root, p, head); e != nil {
		return PullRequestResult{}, e
	}
	dirty, e := selectedText(ctx, root, "", "status", "--porcelain", "--untracked-files=all")
	if e != nil {
		return PullRequestResult{}, e
	}
	if dirty != "" {
		return PullRequestResult{}, errors.New("PR creation requires a clean repository; commit remaining changes first")
	}
	existing, e := lookupPullRequest(ctx, root, p, run)
	if e != nil || existing.URL != "" {
		return existing, e
	}
	// Do not let gh push implicitly: a standalone PR must follow an already
	// reviewed push of this exact head to this exact remote branch.
	remote, _ := selectedText(ctx, root, "", "config", "--get", "branch."+p.Push.Branch+".remote")
	merge, _ := selectedText(ctx, root, "", "config", "--get", "branch."+p.Push.Branch+".merge")
	pushedHead, _ := selectedText(ctx, root, "", "rev-parse", "--verify", "refs/remotes/"+p.Push.Remote+"/"+p.Push.Target)
	if head == "" {
		head = p.head
	}
	if remote != p.Push.Remote || merge != "refs/heads/"+p.Push.Target || pushedHead != head {
		return PullRequestResult{}, errors.New("push this exact commit to the reviewed remote branch before creating a PR")
	}
	// Use the original manual title/body CLI path, with explicit recipient and draft.
	f, e := os.CreateTemp("", "whaleisle-pr-body-*")
	if e != nil {
		return PullRequestResult{}, e
	}
	defer os.Remove(f.Name())
	if _, e = f.WriteString(p.Body); e != nil {
		f.Close()
		return PullRequestResult{}, e
	}
	if e = f.Close(); e != nil {
		return PullRequestResult{}, e
	}
	if e = validatePullRequest(ctx, root, p, head); e != nil {
		return PullRequestResult{}, e
	}
	emitPhase(ctx, "Creating draft PR in "+p.Repository+"…")
	out, e := run(ctx, root, "gh", "pr", "create", "--draft", "--repo", p.Repository, "--base", p.Base, "--head", p.Push.Target, "--title", p.Title, "--body-file", f.Name())
	candidate := firstHTTPURL(strings.TrimSpace(string(out)))
	if e == nil && ValidPullRequestURL(candidate, p.Repository) {
		return PullRequestResult{URL: candidate, Repository: p.Repository}, nil
	}
	// The request may have succeeded remotely. Never issue a second create blindly.
	if ctx.Err() == nil {
		if found, le := lookupPullRequest(ctx, root, p, run); le == nil && found.URL != "" {
			return found, nil
		}
	}
	return PullRequestResult{Repository: p.Repository, Uncertain: true}, errors.New("PR creation outcome is unknown. Check existing PRs before any new create attempt")
}
