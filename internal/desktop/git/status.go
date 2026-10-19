package git

import (
	"context"
	"strconv"
	"strings"
)

// RepositoryStatus is the local titlebar status. No fetch, account lookup or
// change-request request runs as part of a status refresh.
type RepositoryStatus struct {
	Branch, Upstream, DefaultBranch string
	IsRepo, Dirty, HasOrigin        bool
	HasUpstream, IsDefault          bool
	Ahead, Behind, AheadOfDefault   int
	AheadUnreliable                 bool
}

func (s RepositoryStatus) CanPush() bool {
	return s.IsRepo && s.Branch != "" && s.Behind == 0 && (s.Ahead > 0 || s.AheadUnreliable) && (s.HasUpstream || s.HasOrigin)
}

func (s RepositoryStatus) CanCreatePR() bool {
	return s.IsRepo && s.Branch != "" && !s.Dirty && s.Behind == 0 && s.AheadOfDefault > 0 && (s.HasUpstream || s.HasOrigin)
}

func ReadRepositoryStatus(ctx context.Context, root string) (RepositoryStatus, error) {
	var status RepositoryStatus
	if cleanWorkspaceRoot(root) == "" {
		return status, ErrNotRepo
	}
	run := func(args ...string) (string, error) { return selectedText(ctx, root, "", args...) }
	inside, err := run("rev-parse", "--is-inside-work-tree")
	if err != nil || inside != "true" {
		return status, ErrNotRepo
	}
	status.IsRepo = true
	porcelain, err := run("status", "--porcelain=v2", "--branch")
	if err != nil {
		return status, err
	}
	sawCounts := false
	for _, line := range strings.Split(porcelain, "\n") {
		switch {
		case strings.HasPrefix(line, "# branch.head "):
			name := strings.TrimSpace(strings.TrimPrefix(line, "# branch.head "))
			if !strings.HasPrefix(name, "(") {
				status.Branch = name
			}
		case strings.HasPrefix(line, "# branch.upstream "):
			status.Upstream = strings.TrimSpace(strings.TrimPrefix(line, "# branch.upstream "))
		case strings.HasPrefix(line, "# branch.ab "):
			counts := strings.Fields(strings.TrimPrefix(line, "# branch.ab "))
			if len(counts) == 2 {
				status.Ahead, _ = strconv.Atoi(strings.TrimPrefix(counts[0], "+"))
				status.Behind, _ = strconv.Atoi(strings.TrimPrefix(counts[1], "-"))
				sawCounts = true
			}
		case line != "" && !strings.HasPrefix(line, "#"):
			status.Dirty = true
		}
	}
	status.HasUpstream = status.Upstream != "" && sawCounts
	remotesText, _ := run("remote")
	remotes := strings.Fields(remotesText)
	primary := ""
	if len(remotes) > 0 {
		primary = remotes[0]
	}
	for _, remote := range remotes {
		if remote == "origin" {
			status.HasOrigin, primary = true, remote
		}
	}
	defaultBranch := func(remote string) string {
		if remote != "" {
			if ref, e := run("symbolic-ref", "--quiet", "refs/remotes/"+remote+"/HEAD"); e == nil {
				return strings.TrimPrefix(ref, "refs/remotes/"+remote+"/")
			}
		}
		for _, candidate := range []string{"main", "master"} {
			if _, e := run("rev-parse", "--verify", "--quiet", "refs/heads/"+candidate); e == nil {
				return candidate
			}
		}
		return ""
	}
	if status.HasOrigin {
		status.DefaultBranch = defaultBranch(primary)
	} else {
		status.DefaultBranch = defaultBranch("")
	}
	status.IsDefault = status.Branch != "" && (status.Branch == status.DefaultBranch || (status.DefaultBranch == "" && (status.Branch == "main" || status.Branch == "master")))
	if status.Branch != "" && (!status.IsDefault || !status.HasUpstream) {
		configured, _ := run("config", "--get", "branch."+status.Branch+".gh-merge-base")
		baseDefault := status.DefaultBranch
		if !status.HasOrigin && primary != "" {
			baseDefault = defaultBranch(primary)
		}
		base := ""
		for _, candidate := range []string{configured, baseDefault, "main", "master"} {
			candidate = strings.TrimPrefix(candidate, "origin/")
			if primary != "" {
				candidate = strings.TrimPrefix(candidate, primary+"/")
			}
			if candidate == "" || candidate == status.Branch {
				continue
			}
			if primary != "" {
				if _, e := run("show-ref", "--verify", "--quiet", "refs/remotes/"+primary+"/"+candidate); e == nil {
					base = primary + "/" + candidate
					break
				}
			}
			if _, e := run("show-ref", "--verify", "--quiet", "refs/heads/"+candidate); e == nil {
				base = candidate
				break
			}
		}
		count := 0
		if base != "" {
			value, e := run("rev-list", "--count", base+"..HEAD")
			count, err = strconv.Atoi(value)
			status.AheadUnreliable = e != nil || err != nil
			count = max(0, count)
		}
		if !status.HasUpstream {
			status.Ahead, status.Behind = count, 0
		}
		if !status.IsDefault {
			status.AheadOfDefault = count
		}
	}
	if err := ctx.Err(); err != nil {
		return status, err
	}
	return status, nil
}
