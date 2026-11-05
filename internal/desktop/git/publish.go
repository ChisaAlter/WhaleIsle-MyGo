package git

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// IsRepository checks the selected folder without initializing or changing it.
func IsRepository(root string) bool {
	out, err := localCommand(root, "rev-parse", "--is-inside-work-tree")
	return err == nil && out == "true"
}

// Initialize creates an empty main branch only after an explicit Initialize click.
func Initialize(root string) (string, error) {
	root = cleanWorkspaceRoot(root)
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return "", errors.New("select an existing workspace directory")
	}
	if IsRepository(root) {
		return "Repository already initialized", nil
	}
	return localCommand(root, "init", "-b", "main")
}

type PublishInput struct {
	Name, RemoteURL string
	Public          bool
}

var repositoryName = regexp.MustCompile(`^(?:[A-Za-z0-9][A-Za-z0-9_.-]*/)?[A-Za-z0-9][A-Za-z0-9_.-]*$`)
var scpRemote = regexp.MustCompile(`^[A-Za-z0-9._-]+@[A-Za-z0-9.-]+:[^\s]+$`)

func validatePublishInput(root string, input PublishInput) (PublishInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.RemoteURL = strings.TrimSpace(input.RemoteURL)
	if input.RemoteURL != "" {
		if strings.ContainsAny(input.RemoteURL, "\x00\r\n\t ") {
			return input, errors.New("enter one HTTPS or SSH remote URL")
		}
		if scpRemote.MatchString(input.RemoteURL) {
			return input, nil
		}
		u, err := url.Parse(input.RemoteURL)
		if err != nil || (u.Scheme != "https" && u.Scheme != "ssh") || u.Hostname() == "" || u.Path == "" || u.Fragment != "" || u.RawQuery != "" {
			return input, errors.New("use an HTTPS or SSH remote URL")
		}
		if u.User != nil {
			if _, password := u.User.Password(); password || u.Scheme == "https" {
				return input, errors.New("do not put credentials in the remote URL; use your existing Git credential manager")
			}
		}
		return input, nil
	}
	if input.Name == "" {
		input.Name = filepath.Base(cleanWorkspaceRoot(root))
	}
	if !repositoryName.MatchString(input.Name) {
		return input, errors.New("enter a repository name or owner/name")
	}
	return input, nil
}

func OriginURL(root string) (string, error) {
	return localCommand(root, "remote", "get-url", "origin")
}

// PreparePublish validates an explicit review without modifying local or remote data.
func PreparePublish(root string, input PublishInput) (PublishInput, error) {
	if !IsRepository(root) {
		return input, ErrNotRepo
	}
	if _, err := OriginURL(root); err == nil {
		return input, errors.New("origin already exists; use Push instead of publishing again")
	}
	return validatePublishInput(root, input)
}

// PublishRepository matches the original desktop contract. Existing URL adds
// origin; blank URL creates a GitHub repository through the installed gh login.
// Empty repositories get a remote only; no implicit initial commit is created.
func PublishRepository(root string, input PublishInput) (string, error) {
	input, err := PreparePublish(root, input)
	if err != nil {
		return "", err
	}
	_, headErr := localCommand(root, "rev-parse", "--verify", "HEAD")
	hasCommits := headErr == nil
	if input.RemoteURL != "" {
		if _, err := localCommand(root, "remote", "add", "origin", input.RemoteURL); err != nil {
			return "", err
		}
		if !hasCommits {
			return "origin added. Create a commit, then use Push.", nil
		}
		out, err := PushCurrent(root)
		if err != nil {
			return "", fmt.Errorf("origin was added, but push failed; resolve the error and use Push instead of publishing again: %w", err)
		}
		return "origin added and current branch pushed.\n" + out, nil
	}
	if errText := requireGH(nil); errText != "" {
		return "", errors.New(errText)
	}
	visibility := "--private"
	if input.Public {
		visibility = "--public"
	}
	args := []string{"repo", "create", input.Name, visibility, "--source=.", "--remote=origin", "--yes"}
	if hasCommits {
		args = append(args, "--push")
	}
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCommandTimeout)
	defer cancel()
	out, err := combinedExecRunner(ctx, root, "gh", args...)
	if err != nil {
		return "", fmt.Errorf("repository publication failed; creation or remote setup may have partially completed. Check GitHub and origin before retrying: %w: %s", err, strings.TrimSpace(string(out)))
	}
	result := strings.TrimSpace(string(out))
	if !hasCommits {
		result += "\nRepository created and origin added. Create a commit, then use Push."
	}
	return result, nil
}

// RepositoryWebURL converts an exact HTTPS/SSH Git remote to its repository
// landing page. Credentials, query/fragment and non-repository paths are refused.
func RepositoryWebURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if scpRemote.MatchString(raw) {
		parts := strings.SplitN(raw, "@", 2)
		raw = "ssh://" + parts[0] + "@" + strings.Replace(parts[1], ":", "/", 1)
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "ssh") || u.Hostname() == "" || u.RawQuery != "" || u.Fragment != "" || u.RawPath != "" {
		return "", errors.New("invalid repository URL")
	}
	if u.User != nil && (u.Scheme != "ssh" || u.User.String() != "git") {
		return "", errors.New("credential-bearing repository URL is not supported")
	}
	repo := strings.TrimSuffix(strings.Trim(u.Path, "/"), ".git")
	if !githubRepositoryPattern.MatchString(repo) || strings.Contains(repo, "..") {
		return "", errors.New("invalid repository path")
	}
	return "https://" + strings.ToLower(u.Host) + "/" + repo, nil
}

type PublishResult struct{ Output, URL string }

// PublishRepositoryWithResult preserves the existing publication contract and
// provides a validated browser target only after success. It never opens it.
func PublishRepositoryWithResult(root string, input PublishInput) (PublishResult, error) {
	out, err := PublishRepository(root, input)
	result := PublishResult{Output: out}
	if err != nil {
		return result, err
	}
	remote, e := OriginURL(root)
	if e == nil {
		result.URL, _ = RepositoryWebURL(remote)
	}
	if result.URL == "" {
		result.URL, _ = RepositoryWebURL(firstHTTPURL(out))
	}
	return result, nil
}
