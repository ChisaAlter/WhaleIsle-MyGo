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

// PullRequestPlan binds a PR's exact recipient, head/base and manual text.
// Private identity/token fields ensure edited text always requires a fresh review.
type PullRequestPlan struct {
	Repository, Base, Title, Body string
	Host, HeadRepository, Head    string
	Draft                         bool
	Push                          PushPlan
	root, gitDir, head, token     string
}
type PullRequestResult struct {
	URL, Repository     string
	Host                string
	Existing, Uncertain bool
}

var githubRepositoryPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

func githubRepository(remote string) (string, error) {
	_, repo, err := repositoryIdentity(remote)
	return repo, err
}
func repositoryIdentity(remote string) (host, repo string, err error) {
	target, e := RepositoryWebURL(remote)
	if e != nil {
		return "", "", e
	}
	u, e := url.Parse(target)
	if e != nil {
		return "", "", e
	}
	return strings.ToLower(u.Host), strings.Trim(u.Path, "/"), nil
}
func githubCLIRepository(host, repo string) string {
	if host == "" || strings.EqualFold(host, "github.com") {
		return repo
	}
	return host + "/" + repo
}
func ValidPullRequestURL(raw, repository string) bool {
	return ValidPullRequestURLForHost(raw, "github.com", repository)
}
func ValidPullRequestURLForHost(raw, host, repository string) bool {
	if host == "" {
		host = "github.com"
	}
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || !strings.EqualFold(u.Host, host) || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.RawPath != "" || !githubRepositoryPattern.MatchString(repository) {
		return false
	}
	prefix := "/" + repository + "/pull/"
	if !strings.HasPrefix(strings.ToLower(u.Path), strings.ToLower(prefix)) {
		return false
	}
	n, e := strconv.Atoi(u.Path[len(prefix):])
	return e == nil && n > 0
}

type selectedPRRepository struct {
	NameWithOwner    string `json:"nameWithOwner"`
	URL              string `json:"url"`
	DefaultBranchRef struct {
		Name string `json:"name"`
	} `json:"defaultBranchRef"`
}

// Omitting a repository argument is deliberate: gh's selected upstream/base
// repository may differ from the fork receiving the reviewed push.
func resolvePRRepository(ctx context.Context, root string, run combinedRunner) (selectedPRRepository, string, error) {
	var repo selectedPRRepository
	out, e := run(ctx, root, "gh", "repo", "view", "--json", "nameWithOwner,url,defaultBranchRef")
	if e != nil {
		return repo, "", fmt.Errorf("cannot resolve selected PR repository: %w: %s", e, strings.TrimSpace(string(out)))
	}
	if e = json.Unmarshal(out, &repo); e != nil {
		return repo, "", e
	}
	host, name, e := repositoryIdentity(repo.URL)
	if e != nil || !strings.EqualFold(name, repo.NameWithOwner) {
		return repo, "", errors.New("GitHub did not return an unambiguous selected repository")
	}
	return repo, host, nil
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
	data, _ := json.Marshal([]any{p.Host, p.HeadRepository, p.Head, p.Repository, p.Base, p.Title, p.Body, p.Draft, p.Push, p.root, p.gitDir, p.head})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
func PreparePullRequest(ctx context.Context, root string, push PushPlan, title, body string) (PullRequestPlan, error) {
	return preparePullRequest(ctx, root, push, title, body, combinedExecRunner)
}

// PreparePullRequestMode includes the user's selected draft/ready state in review.
func PreparePullRequestMode(ctx context.Context, root string, push PushPlan, title, body string, draft bool) (PullRequestPlan, error) {
	return preparePullRequestMode(ctx, root, push, title, body, draft, combinedExecRunner)
}
func preparePullRequest(ctx context.Context, root string, push PushPlan, title, body string, run combinedRunner) (PullRequestPlan, error) {
	return preparePullRequestMode(ctx, root, push, title, body, true, run)
}
func PreparePullRequestForNewBranch(ctx context.Context, root string, push PushPlan, title, body string, draft bool, newBranch string) (PullRequestPlan, error) {
	return preparePullRequestBranch(ctx, root, push, title, body, draft, newBranch, combinedExecRunner)
}
func preparePullRequestMode(ctx context.Context, root string, push PushPlan, title, body string, draft bool, run combinedRunner) (PullRequestPlan, error) {
	return preparePullRequestBranch(ctx, root, push, title, body, draft, "", run)
}
func preparePullRequestBranch(ctx context.Context, root string, push PushPlan, title, body string, draft bool, newBranch string, run combinedRunner) (p PullRequestPlan, err error) {
	p.Draft = draft
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
	current, e := reviewedBranchPush(root, newBranch)
	if e != nil {
		return p, e
	}
	if current != push {
		return p, errors.New("push destination changed; review again")
	}
	pushHost, headRepository, e := repositoryIdentity(push.URL)
	if e != nil {
		return p, e
	}
	repo, host, e := resolvePRRepository(ctx, root, run)
	if e != nil {
		return p, e
	}
	if !strings.EqualFold(pushHost, host) {
		return p, errors.New("push and selected PR repository belong to different GitHub hosts")
	}
	p.Host, p.Repository, p.HeadRepository = host, repo.NameWithOwner, headRepository
	p.Head = push.Target
	if !strings.EqualFold(p.Repository, p.HeadRepository) {
		p.Head = strings.SplitN(p.HeadRepository, "/", 2)[0] + ":" + push.Target
	}
	p.Base, _ = selectedText(ctx, root, "", "config", "--get", "branch."+push.Branch+".gh-merge-base")
	if p.Base != "" {
		p.Base = strings.TrimPrefix(p.Base, "origin/")
	}
	if p.Base == "" {
		remote, _ := selectedText(ctx, root, "", "config", "--get", "branch."+push.Branch+".remote")
		merge, _ := selectedText(ctx, root, "", "config", "--get", "branch."+push.Branch+".merge")
		remoteURL, _ := selectedText(ctx, root, "", "remote", "get-url", remote)
		upstreamHost, upstreamRepo, _ := repositoryIdentity(remoteURL)
		upstream := strings.TrimPrefix(merge, "refs/heads/")
		if strings.EqualFold(upstreamHost, p.Host) && strings.EqualFold(upstreamRepo, p.Repository) && upstream != "" && upstream != push.Target {
			p.Base = upstream
		}
	}
	if p.Base == "" {
		p.Base = repo.DefaultBranchRef.Name
	}
	if p.Base == "" || checkBranchName(root, p.Base) != nil {
		return p, errors.New("GitHub did not return a valid PR base branch")
	}
	if p.Base == push.Target && strings.EqualFold(p.Repository, p.HeadRepository) {
		return p, errors.New("create a feature branch before opening a PR against the same branch")
	}
	p.token = pullRequestToken(p)
	return
}
func validatePullRequest(ctx context.Context, root string, p PullRequestPlan, expectedHead string) error {
	return validatePullRequestBeforeCommit(ctx, root, p, expectedHead, "")
}
func validatePullRequestBeforeCommit(ctx context.Context, root string, p PullRequestPlan, expectedHead, newBranch string) error {
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
	current, e := reviewedBranchPush(root, newBranch)
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
	out, e := run(ctx, root, "gh", "pr", "list", "--repo", githubCLIRepository(p.Host, p.Repository), "--head", p.Head, "--base", p.Base, "--state", "open", "--json", "number,title,url,state,headRefName,baseRefName,headRepository,headRepositoryOwner")
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
	if pinfo.State != "OPEN" || !ValidPullRequestURLForHost(pinfo.URL, p.Host, p.Repository) || pinfo.HeadRefName != p.Push.Target || pinfo.BaseRefName != p.Base || !strings.EqualFold(pinfo.HeadRepositoryOwner.Login+"/"+pinfo.HeadRepository.Name, p.HeadRepository) {
		return PullRequestResult{}, errors.New("GitHub returned an unexpected PR URL or state")
	}
	return PullRequestResult{URL: pinfo.URL, Repository: p.Repository, Host: p.Host, Existing: true}, nil
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
	// Use the original manual title/body CLI path, with explicit recipient and reviewed draft/ready state.
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
	emitPhase(ctx, "Creating PR in "+p.Repository+"…")
	args := []string{"pr", "create"}
	if p.Draft {
		args = append(args, "--draft")
	}
	args = append(args, "--repo", githubCLIRepository(p.Host, p.Repository), "--base", p.Base, "--head", p.Head, "--title", p.Title, "--body-file", f.Name())
	out, e := run(ctx, root, "gh", args...)
	candidate := firstHTTPURL(strings.TrimSpace(string(out)))
	if e == nil && ValidPullRequestURLForHost(candidate, p.Host, p.Repository) {
		return PullRequestResult{URL: candidate, Repository: p.Repository, Host: p.Host}, nil
	}
	// The request may have succeeded remotely. Never issue a second create blindly.
	if ctx.Err() == nil {
		if found, le := lookupPullRequest(ctx, root, p, run); le == nil && found.URL != "" {
			return found, nil
		}
	}
	return PullRequestResult{Repository: p.Repository, Host: p.Host, Uncertain: true}, errors.New("PR creation outcome is unknown. Check existing PRs before any new create attempt")
}
