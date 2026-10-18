package git

import (
	"errors"
	"fmt"
	"strings"
)

// PushPlan binds the explicit review to a branch, destination and one push URL.
type PushPlan struct {
	Branch, Remote, Target, URL string
	DefaultBranch               bool
}

func PreparePush(root string) (PushPlan, error) {
	branch, err := localCommand(root, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil || branch == "" {
		return PushPlan{}, errors.New("cannot push from detached HEAD")
	}
	return preparePushBranch(root, branch)
}

// PreparePushForNewBranch previews an explicit destination without checking out
// or creating the branch. The same destination is rechecked after commit.
func PreparePushForNewBranch(root, branch string) (PushPlan, error) {
	if err := checkBranchName(root, branch); err != nil {
		return PushPlan{}, err
	}
	if _, err := localCommand(root, "show-ref", "--verify", "--quiet", "refs/heads/"+branch); err == nil {
		return PushPlan{}, errors.New("new branch already exists; review again")
	}
	return preparePushBranch(root, branch)
}
func preparePushBranch(root, branch string) (PushPlan, error) {
	if err := checkBranchName(root, branch); err != nil {
		return PushPlan{}, err
	}
	remotesText, err := localCommand(root, "remote")
	if err != nil {
		return PushPlan{}, err
	}
	remotes := strings.Fields(remotesText)
	remote, _ := localCommand(root, "config", "--get", "branch."+branch+".remote")
	merge, _ := localCommand(root, "config", "--get", "branch."+branch+".merge")
	target := branch
	known := false
	for _, name := range remotes {
		if name == remote {
			known = true
		}
	}
	if known && strings.HasPrefix(merge, "refs/heads/") {
		target = strings.TrimPrefix(merge, "refs/heads/")
	} else {
		remote = ""
		for _, name := range remotes {
			if name == "origin" {
				remote = name
				break
			}
		}
		if remote == "" && len(remotes) == 1 {
			remote = remotes[0]
		}
	}
	if remote == "" || strings.HasPrefix(remote, "-") {
		return PushPlan{}, errors.New("configure one unambiguous push remote, or publish this repository first")
	}
	if err := checkBranchName(root, target); err != nil {
		return PushPlan{}, err
	}
	urls, err := localCommand(root, "remote", "get-url", "--push", "--all", remote)
	if err != nil {
		return PushPlan{}, err
	}
	list := strings.Split(strings.TrimSpace(urls), "\n")
	if len(list) != 1 || strings.TrimSpace(list[0]) == "" {
		return PushPlan{}, errors.New("multiple push destinations require explicit Git configuration before using this action")
	}
	defaultRef, _ := localCommand(root, "symbolic-ref", "--quiet", "refs/remotes/"+remote+"/HEAD")
	isDefault := defaultRef == "refs/remotes/"+remote+"/"+target
	if defaultRef == "" {
		isDefault = target == "main" || target == "master"
	}
	return PushPlan{Branch: branch, Remote: remote, Target: target, URL: strings.TrimSpace(list[0]), DefaultBranch: isDefault}, nil
}

// PushCurrent establishes an upstream on the explicitly selected current branch.
func PushCurrent(root string) (string, error) {
	plan, err := PreparePush(root)
	if err != nil {
		return "", err
	}
	return PushReviewed(root, plan)
}

// PushReviewed rechecks its destination immediately before an ordinary, non-force push.
func PushReviewed(root string, reviewed PushPlan) (string, error) {
	current, err := PreparePush(root)
	if err != nil {
		return "", err
	}
	if current != reviewed {
		return "", errors.New("branch or push destination changed; review again")
	}
	return runGitCommand(root, DefaultCommandTimeout, combinedExecRunner, "push", "--set-upstream", current.Remote, "HEAD:refs/heads/"+current.Target)
}

type CommitPushResult struct {
	Output    string
	Committed bool
}

// CommitPushReviewed never stages files and never repeats a successful commit
// after a push failure. The caller can offer the separate Push action to retry.
func CommitPushReviewed(root, message, token string, plan PushPlan) (CommitPushResult, error) {
	current, err := PreparePush(root)
	if err != nil {
		return CommitPushResult{}, err
	}
	if current != plan {
		return CommitPushResult{}, errors.New("branch or push destination changed; review again")
	}
	out, err := CommitReviewed(root, message, token)
	if err != nil {
		return CommitPushResult{}, err
	}
	result := CommitPushResult{Output: out, Committed: true}
	pushed, err := PushReviewed(root, plan)
	if err != nil {
		return result, fmt.Errorf("committed locally, but push failed. Use Push to retry without making another commit: %w", err)
	}
	result.Output += "\n" + pushed
	return result, nil
}
