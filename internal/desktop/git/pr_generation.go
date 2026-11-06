package git

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"whaleisle/internal/credentials"
	"whaleisle/internal/paths"
)

type PRGenerationInput struct{ CommitSummary, DiffSummary, DiffPatch, FallbackTitle, Template string }
type GeneratedPRContent struct {
	Title                  string `json:"title"`
	Body                   string `json:"body"`
	Source                 string `json:"-"`
	Repository, Base, Head string `json:"-"`
}

var prCommitPrefix = regexp.MustCompile(`(?i)^[0-9a-f]{7,}\s+`)

func generatePRText(ctx context.Context, in PRGenerationInput, key, base string, client *http.Client) (GeneratedPRContent, error) {
	if e := ctx.Err(); e != nil {
		return GeneratedPRContent{}, e
	}
	if key == "" {
		title := in.FallbackTitle
		for _, line := range strings.Split(in.CommitSummary, "\n") {
			if strings.TrimSpace(line) != "" {
				title = prCommitPrefix.ReplaceAllString(line, "")
				break
			}
		}
		return GeneratedPRContent{Title: generatedLine(title), Body: strings.TrimSpace(in.CommitSummary), Source: "heuristic"}, nil
	}
	system := `You write pull request copy. Reply with JSON only: {"title":"...","body":"..."}. Title is imperative, <=72 chars. Body is markdown.`
	user := ""
	if in.Template != "" {
		user = "Follow this pull request template:\n" + generationLimit(in.Template, 8000) + "\n\n"
	}
	user += "Commits:\n" + generationLimit(in.CommitSummary, 20000) + "\n\nDiff stat:\n" + generationLimit(in.DiffSummary, 20000) + "\n\nPatch:\n" + generationLimit(in.DiffPatch, 60000)
	failure := "pull request content generation failed; retry or keep editing manually"
	data, e := requestGeneratedJSON(ctx, system, user, key, base, failure, client)
	if e != nil {
		return GeneratedPRContent{}, e
	}
	var g GeneratedPRContent
	if json.Unmarshal(data, &g) != nil || (strings.TrimSpace(g.Title) == "" && strings.TrimSpace(g.Body) == "") {
		return GeneratedPRContent{}, errors.New(failure)
	}
	if strings.TrimSpace(g.Title) == "" {
		g.Title = in.FallbackTitle
	}
	g.Title = generatedLine(g.Title)
	g.Body = strings.TrimSpace(g.Body)
	g.Source = "model"
	return g, nil
}

// GeneratePullRequestContent uses the original committed branch range and base
// template. It never fetches, pushes, creates a PR, or modifies the draft.
func GeneratePullRequestContent(ctx context.Context, root string) (GeneratedPRContent, error) {
	push, e := PreparePush(root)
	if e != nil {
		return GeneratedPRContent{}, e
	}
	// A local placeholder is only for resolving the original target/base context;
	// it is not a create request and is never shown as generated copy.
	plan, e := PreparePullRequest(ctx, root, push, push.Branch, "")
	if e != nil {
		return GeneratedPRContent{}, e
	}
	in, baseSHA, e := collectPRGenerationInput(ctx, root, plan)
	if e != nil {
		return GeneratedPRContent{}, e
	}
	creds, e := credentials.Load(paths.UserDataDir())
	if e != nil {
		return GeneratedPRContent{}, errors.New("cannot read saved generation credentials; keep editing PR text manually")
	}
	emitPhase(ctx, "Generating PR title and body…")
	g, e := generatePRText(ctx, in, creds.APIKey, creds.BaseURL, nil)
	if e != nil {
		return g, e
	}
	if e = validatePullRequest(ctx, root, plan, plan.head); e != nil {
		return GeneratedPRContent{}, e
	}
	// Resolve again, so a concurrent fetch changing the base cannot silently
	// attach copy generated from a superseded range to a fresh review.
	_, currentBase, e := resolvePRGenerationBase(ctx, root, plan)
	if e != nil {
		return GeneratedPRContent{}, e
	}
	if currentBase != baseSHA {
		return GeneratedPRContent{}, errors.New("PR base changed during generation; generate again")
	}
	g.Repository, g.Base, g.Head = plan.Repository, plan.Base, plan.head
	return g, nil
}
func resolvePRGenerationBase(ctx context.Context, root string, p PullRequestPlan) (string, string, error) {
	remotes, e := selectedText(ctx, root, "", "remote")
	if e != nil {
		return "", "", e
	}
	for _, remote := range strings.Fields(remotes) {
		url, e := selectedText(ctx, root, "", "remote", "get-url", remote)
		if e != nil {
			continue
		}
		host, repo, e := repositoryIdentity(url)
		if e != nil || !strings.EqualFold(host, p.Host) || !strings.EqualFold(repo, p.Repository) {
			continue
		}
		ref := "refs/remotes/" + remote + "/" + p.Base
		sha, e := selectedText(ctx, root, "", "rev-parse", "--verify", ref+"^{commit}")
		if e == nil {
			return ref, sha, nil
		}
	}
	return "", "", errors.New("fetch the PR target base branch before generating content; no matching local target ref is available")
}
func collectPRGenerationInput(ctx context.Context, root string, p PullRequestPlan) (in PRGenerationInput, baseSHA string, err error) {
	if err = validatePullRequest(ctx, root, p, p.head); err != nil {
		return
	}
	_, baseSHA, err = resolvePRGenerationBase(ctx, root, p)
	if err != nil {
		return
	}
	// Freeze both range endpoints; no worktree or unselected file is sent.
	rangeSpec := baseSHA + ".." + p.head
	in.FallbackTitle = p.Push.Branch
	in.CommitSummary, err = generationGitText(ctx, root, 20000, "log", "--oneline", rangeSpec)
	if err != nil {
		return
	}
	// PRs compare the branch with its merge base. A two-dot diff would include
	// unrelated changes made on the target branch as apparent deletions.
	diffRange := baseSHA + "..." + p.head
	in.DiffSummary, err = generationGitText(ctx, root, 20000, "diff", "--no-ext-diff", "--no-textconv", "--stat", diffRange)
	if err != nil {
		return
	}
	in.DiffPatch, err = generationGitText(ctx, root, 60000, "diff", "--no-ext-diff", "--no-textconv", "--patch", "--minimal", diffRange)
	if err != nil {
		return
	}
	in.Template = readPRGenerationTemplate(ctx, root, baseSHA)
	return
}
