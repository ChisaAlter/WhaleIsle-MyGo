package git

// Port of WhaleIsle@90465b67 src/main/git-generate.js. This uses its existing
// DeepSeek REST contract, not an invented Harness RPC or generic LLM endpoint.
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"whaleisle/internal/credentials"
	"whaleisle/internal/paths"
)

type CommitGenerationInput struct {
	Summary, Patch, RecentSubjects string
	IncludeBranch                  bool
}
type GeneratedCommitMessage struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
	Branch  string `json:"branch"`
	Source  string `json:"-"`
}

func (g GeneratedCommitMessage) Text() string {
	if g.Body == "" {
		return g.Subject
	}
	return g.Subject + "\n\n" + g.Body
}
func generatedLine(s string) string {
	s = strings.TrimSpace(strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")[0])
	s = strings.TrimSpace(strings.TrimRight(s, "."))
	r := []rune(s)
	if len(r) > 72 {
		s = strings.TrimSpace(string(r[:72]))
	}
	if s == "" {
		return "Update project files"
	}
	return s
}
func generationLimit(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "\n\n[truncated]"
}
func heuristicCommit(summary string) GeneratedCommitMessage {
	type row struct{ status, path string }
	var rows []row
	for _, line := range strings.Split(summary, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		r := row{"M", line}
		if len(parts) > 1 {
			r = row{parts[0], parts[len(parts)-1]}
		}
		rows = append(rows, r)
	}
	g := GeneratedCommitMessage{Subject: "Update project files", Source: "heuristic"}
	if len(rows) == 0 {
		return g
	}
	verb := "Update"
	switch rows[0].status[0] {
	case 'A':
		verb = "Add"
	case 'D':
		verb = "Remove"
	case 'R':
		verb = "Rename"
	}
	g.Subject = verb + " " + rows[0].path
	if len(rows) > 1 {
		g.Subject += fmt.Sprintf(" and %d other files", len(rows)-1)
	}
	var body []string
	for _, r := range rows {
		body = append(body, r.status+"\t"+r.path)
	}
	g.Subject = generatedLine(g.Subject)
	g.Body = strings.Join(body, "\n")
	return g
}

// GenerateSelectedCommitMessage reads the same desktop credentials used by the
// original. No key => deterministic name-status heuristic. Configured model
// failure => error with manual-entry/retry available; no silent fake AI output.
func GenerateSelectedCommitMessage(ctx context.Context, root string, p SelectedCommitPlan) (GeneratedCommitMessage, error) {
	canonical, canonicalErr := filepath.Abs(root)
	if canonicalErr == nil {
		canonical, canonicalErr = filepath.EvalSymlinks(canonical)
	}
	if p.token == "" || canonicalErr != nil || canonical != p.root {
		return GeneratedCommitMessage{}, errors.New("review selected files before generating a message")
	}
	creds, e := credentials.Load(paths.UserDataDir())
	if e != nil {
		return GeneratedCommitMessage{}, errors.New("cannot read saved generation credentials; enter a message manually")
	}
	recent, recentErr := selectedText(ctx, root, "", "log", "-n", "8", "--pretty=%s")
	if recentErr != nil {
		recent = ""
	}
	return generateCommitText(ctx, CommitGenerationInput{Summary: p.Summary, Patch: p.Patch, RecentSubjects: recent, IncludeBranch: p.NewBranch != ""}, creds.APIKey, creds.BaseURL, nil)
}
func generateCommitText(ctx context.Context, in CommitGenerationInput, key, base string, client *http.Client) (GeneratedCommitMessage, error) {
	if e := ctx.Err(); e != nil {
		return GeneratedCommitMessage{}, e
	}
	if key == "" {
		return heuristicCommit(in.Summary), nil
	}
	branch := ""
	if in.IncludeBranch {
		branch = `,"branch":"feature/..."`
	}
	system := `You write git commit messages. Reply with JSON only: {"subject":"...","body":"..."` + branch + `}. Subject is imperative, <=72 chars, no trailing period. Body may be empty.`
	if in.IncludeBranch {
		system += ` Also include "branch":"feature/short-kebab-name".`
	}
	user := ""
	if in.RecentSubjects != "" {
		user = "Recent commit subjects:\n" + generationLimit(in.RecentSubjects, 2000) + "\n\n"
	}
	user += "Staged name-status:\n" + generationLimit(in.Summary, 8000) + "\n\nStaged patch:\n" + generationLimit(in.Patch, 50000)
	failure := errors.New("commit message generation failed; retry or enter a message manually")
	data, e := requestGeneratedJSON(ctx, system, user, key, base, failure.Error(), client)
	if e != nil {
		return GeneratedCommitMessage{}, e
	}
	var result GeneratedCommitMessage
	if json.Unmarshal(data, &result) != nil || (strings.TrimSpace(result.Subject) == "" && strings.TrimSpace(result.Body) == "") {
		return result, failure
	}
	result.Subject = generatedLine(result.Subject)
	result.Body = strings.TrimSpace(result.Body)
	result.Branch = strings.TrimSpace(result.Branch)
	result.Source = "model"
	return result, nil
}
