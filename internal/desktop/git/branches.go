package git

import (
	"errors"
	"sort"
	"strings"
)

// Branch is an existing local or remote-tracking ref, never an arbitrary revision.
type Branch struct {
	Name, LocalName string
	Remote, Current bool
}

func Branches(root string) ([]Branch, error) {
	out, err := localCommand(root, "for-each-ref", "--format=%(refname)%09%(HEAD)%09%(symref)%09.", "refs/heads", "refs/remotes")
	if err != nil {
		return nil, err
	}
	return parseBranches(out), nil
}

func parseBranches(out string) []Branch {
	branches := []Branch{}
	locals := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) < 2 || (len(fields) > 2 && fields[2] != "") {
			continue
		}
		ref := fields[0]
		branch := Branch{Current: fields[1] == "*"}
		switch {
		case strings.HasPrefix(ref, "refs/heads/"):
			branch.Name = strings.TrimPrefix(ref, "refs/heads/")
			branch.LocalName = branch.Name
			locals[branch.Name] = true
		case strings.HasPrefix(ref, "refs/remotes/"):
			branch.Name = strings.TrimPrefix(ref, "refs/remotes/")
			at := strings.IndexByte(branch.Name, '/')
			if at < 1 || at == len(branch.Name)-1 || strings.HasSuffix(branch.Name, "/HEAD") {
				continue
			}
			branch.Remote = true
			branch.LocalName = branch.Name[at+1:]
		default:
			continue
		}
		if branch.Name != "" {
			branches = append(branches, branch)
		}
	}
	visible := branches[:0]
	for _, branch := range branches {
		if branch.Remote && strings.HasPrefix(branch.Name, "origin/") && locals[branch.LocalName] {
			continue
		}
		visible = append(visible, branch)
	}
	sort.SliceStable(visible, func(i, j int) bool {
		a, b := visible[i], visible[j]
		if a.Current != b.Current {
			return a.Current
		}
		if a.Remote != b.Remote {
			return !a.Remote
		}
		return a.Name < b.Name
	})
	return visible
}

func checkBranchName(root, name string) error {
	if name == "" || strings.TrimSpace(name) != name || strings.HasPrefix(name, "-") || strings.ContainsAny(name, "\x00\r\n\t") {
		return errors.New("enter a valid branch name")
	}
	checked, err := localCommand(root, "check-ref-format", "--branch", name)
	if err != nil {
		return err
	}
	if checked != name {
		return errors.New("enter a literal branch name, not a revision expression")
	}
	return nil
}

// SwitchBranch resolves a listed ref again before switching. Git's normal dirty
// tree and other-worktree protections remain enabled; there is no force/reset.
func SwitchBranch(root, name string) (string, error) {
	if err := checkBranchName(root, name); err != nil {
		return "", err
	}
	branches, err := Branches(root)
	if err != nil {
		return "", err
	}
	for _, branch := range branches {
		if branch.Name != name {
			continue
		}
		if branch.Current {
			return name, nil
		}
		if branch.Remote {
			if err := checkBranchName(root, branch.LocalName); err != nil {
				return "", err
			}
			return localCommand(root, "switch", "--track", "-c", branch.LocalName, name)
		}
		return localCommand(root, "switch", "--no-guess", name)
	}
	return "", errors.New("branch is no longer available; refresh the branch list")
}

// CreateBranch creates and checks out a new local branch without replacing any ref.
func CreateBranch(root, name string) (string, error) {
	if err := checkBranchName(root, name); err != nil {
		return "", err
	}
	return localCommand(root, "switch", "-c", name)
}
