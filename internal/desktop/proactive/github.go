package proactive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"whaleisle/internal/processutil"
)

// GitHubReader uses the installed CLI's credential handling. It neither reads
// token files nor extracts credentials. Only GET repository issue pages run.
type GitHubReader struct{ Executable string }

func (r GitHubReader) Read(ctx context.Context, source Source) ([]SourceItem, error) {
	repo, err := NormalizeRepository(source.Repository)
	if err != nil {
		return nil, err
	}
	query := url.Values{"state": {"open"}, "per_page": {"100"}, "sort": {"updated"}, "direction": {"desc"}}
	if source.LastSuccess != "" {
		since, err := time.Parse(time.RFC3339Nano, source.LastSuccess)
		if err != nil {
			return nil, err
		}
		query.Set("state", "all")
		query.Set("since", since.Add(-time.Second).UTC().Format(time.RFC3339))
	}
	cli := r.Executable
	if cli == "" {
		cli = "gh"
	}
	// GitHub redirects old repository names after a rename. Resolve the selected
	// repository's metadata before accepting item links from its canonical name.
	metadata := exec.CommandContext(ctx, cli, "api", "--hostname", "github.com", "--method", "GET", "repos/"+repo)
	processutil.Background(metadata)
	metadata.Env = githubEnvironment()
	raw, err := metadata.Output()
	if err != nil {
		return nil, fmt.Errorf("GitHub repository lookup failed; check gh sign-in and repository access")
	}
	var identity struct {
		ID       json.Number `json:"id"`
		FullName string      `json:"full_name"`
	}
	if json.Unmarshal(raw, &identity) != nil || identity.ID == "" {
		return nil, fmt.Errorf("GitHub returned an invalid repository identity")
	}
	canonical, err := NormalizeRepository(identity.FullName)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, cli, "api", "--hostname", "github.com", "--method", "GET", "--paginate", "repos/"+repo+"/issues?"+query.Encode())
	processutil.Background(cmd)
	// Disable CLI prompts/pagers. The existing CLI credential provider remains
	// responsible for authentication, including private repository access.
	cmd.Env = githubEnvironment()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("GitHub CLI could not start; install gh and sign in through its account settings")
	}
	items, decodeErr := decodeGitHubPages(stdout, canonical)
	if decodeErr != nil {
		_ = cmd.Process.Kill()
	}
	waitErr := cmd.Wait()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if waitErr != nil {
		return nil, fmt.Errorf("GitHub request failed; check gh sign-in, repository access and network")
	}
	if decodeErr != nil {
		return nil, decodeErr
	}
	for i := range items {
		items[i].Repository = repo
	}
	return items, nil
}

func githubEnvironment() []string {
	return append(os.Environ(), "GH_PROMPT_DISABLED=1", "GH_PAGER=cat", "GH_BROWSER=", "GH_DEBUG=")
}

func decodeGitHubPages(reader io.Reader, repository string) ([]SourceItem, error) {
	decoder := json.NewDecoder(reader)
	var result []SourceItem
	for {
		var page []struct {
			ID          json.Number     `json:"id"`
			Number      int64           `json:"number"`
			Title       string          `json:"title"`
			Body        string          `json:"body"`
			HTMLURL     string          `json:"html_url"`
			State       string          `json:"state"`
			UpdatedAt   string          `json:"updated_at"`
			PullRequest json.RawMessage `json:"pull_request"`
			Labels      []struct {
				Name string `json:"name"`
			} `json:"labels"`
		}
		if err := decoder.Decode(&page); err != nil {
			if errors.Is(err, io.EOF) {
				return result, nil
			}
			return nil, fmt.Errorf("GitHub returned an unreadable issue page")
		}
		for _, row := range page {
			kind, segment := "issue", "issues"
			if len(row.PullRequest) > 0 && string(row.PullRequest) != "null" {
				kind, segment = "pull_request", "pull"
			}
			parsed, err := url.Parse(row.HTMLURL)
			want := "/" + repository + "/" + segment + "/" + strconv.FormatInt(row.Number, 10)
			if err != nil || parsed.Scheme != "https" || parsed.Host != "github.com" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || !strings.EqualFold(parsed.Path, want) || parsed.RawPath != "" || row.ID == "" || row.Number <= 0 {
				return nil, fmt.Errorf("GitHub returned an item outside the selected repository")
			}
			item := SourceItem{ID: row.ID.String(), Repository: repository, Number: row.Number, Title: row.Title, Description: row.Body, URL: row.HTMLURL, Kind: kind, State: row.State, UpdatedAt: row.UpdatedAt}
			for _, label := range row.Labels {
				item.Labels = append(item.Labels, label.Name)
			}
			result = append(result, item)
		}
	}
}
