package git

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPRGenerationOriginalHeuristic(t *testing.T) {
	g, e := generatePRText(context.Background(), PRGenerationInput{CommitSummary: "abc1234 Add review flow.\ndef5678 Keep index", FallbackTitle: "feature/review"}, "", "", nil)
	if e != nil || g.Title != "Add review flow" || g.Body != "abc1234 Add review flow.\ndef5678 Keep index" || g.Source != "heuristic" {
		t.Fatal(g, e)
	}
}
func TestPRGenerationOriginalRequest(t *testing.T) {
	client := &http.Client{Transport: generationTransport(func(r *http.Request) (*http.Response, error) {
		var body struct {
			Model       string
			Temperature float64
			Messages    []struct{ Role, Content string }
		}
		if e := json.NewDecoder(r.Body).Decode(&body); e != nil {
			t.Fatal(e)
		}
		if r.URL.String() != "https://example.invalid/v1/chat/completions" || body.Model != "deepseek-chat" || body.Temperature != 0.2 || len(body.Messages) != 2 || body.Messages[0].Content != `You write pull request copy. Reply with JSON only: {"title":"...","body":"..."}. Title is imperative, <=72 chars. Body is markdown.` || body.Messages[1].Content != "Follow this pull request template:\n## Why\n\nCommits:\nabc1234 Add review\n\nDiff stat:\na | 1 +\n\nPatch:\n+review" {
			t.Fatal(r.URL, body)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"choices":[{"message":{"content":"{\"title\":\"Add review flow.\",\"body\":\"## Why\\nSafe\"}"}}]}`)), Header: make(http.Header)}, nil
	})}
	g, e := generatePRText(context.Background(), PRGenerationInput{CommitSummary: "abc1234 Add review", DiffSummary: "a | 1 +", DiffPatch: "+review", Template: "## Why", FallbackTitle: "feature/review"}, "fixture-only", "https://example.invalid", client)
	if e != nil || g.Title != "Add review flow" || g.Body != "## Why\nSafe" || g.Source != "model" {
		t.Fatal(g, e)
	}
}
func TestPRGenerationFailureAndCancelNeverReturnFallback(t *testing.T) {
	client := &http.Client{Transport: generationTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"choices":[{"message":{"content":"{}"}}]}`)), Header: make(http.Header)}, nil
	})}
	if _, e := generatePRText(context.Background(), PRGenerationInput{CommitSummary: "abc1234 Fallback"}, "fixture", "https://example.invalid", client); e == nil {
		t.Fatal("configured failure fell back")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := generatePRText(ctx, PRGenerationInput{}, "", "", nil); e == nil {
		t.Fatal("canceled heuristic returned")
	}
}
func TestPRGenerationReadsFrozenBranchRangeAndCommittedTemplate(t *testing.T) {
	root, run, write := selectionRepo(t)
	write("PULL_REQUEST_TEMPLATE.md", "## Committed base template")
	run("add", "PULL_REQUEST_TEMPLATE.md")
	run("commit", "-m", "Base template")
	run("remote", "add", "origin", "https://github.com/example/project.git")
	run("update-ref", "refs/remotes/origin/main", "HEAD")
	run("switch", "-c", "feature/review")
	write("a.txt", "committed change\n")
	run("add", "a.txt")
	run("commit", "-m", "Add reviewed behavior")
	write("PULL_REQUEST_TEMPLATE.md", "PRIVATE WORKTREE TEMPLATE")
	write("b.txt", "PRIVATE UNCOMMITTED CHANGE")
	push, _ := PreparePush(root)
	fake := func(context.Context, string, string, ...string) ([]byte, error) {
		return []byte(`{"defaultBranchRef":{"name":"main"}}`), nil
	}
	plan, e := preparePullRequest(context.Background(), root, push, "Temporary title", "", fake)
	if e != nil {
		t.Fatal(e)
	}
	in, base, e := collectPRGenerationInput(context.Background(), root, plan)
	if e != nil || base == "" || in.Template != "## Committed base template" || !strings.Contains(in.CommitSummary, "Add reviewed behavior") || !strings.Contains(in.DiffPatch, "+committed change") || strings.Contains(in.DiffPatch, "PRIVATE") {
		t.Fatal(in, e)
	}
	run("update-ref", "-d", "refs/remotes/origin/main")
	if _, _, e = collectPRGenerationInput(context.Background(), root, plan); e == nil {
		t.Fatal("unfetched target silently replaced")
	}
}
func TestPRGenerationTemplateSkipsSymlinksAndAmbiguousDirectory(t *testing.T) {
	root, run, write := selectionRepo(t)
	if e := os.MkdirAll(filepath.Join(root, ".github", "PULL_REQUEST_TEMPLATE"), 0700); e != nil {
		t.Fatal(e)
	}
	write(".github/PULL_REQUEST_TEMPLATE/a.md", "A")
	write(".github/PULL_REQUEST_TEMPLATE/b.md", "B")
	if e := os.Symlink("../outside-target", filepath.Join(root, "PULL_REQUEST_TEMPLATE.md")); e != nil {
		t.Skip("fixture symlink unavailable")
	}
	run("add", ".")
	run("commit", "-m", "Template fixtures")
	if got := readPRGenerationTemplate(context.Background(), root, "HEAD"); got != "" {
		t.Fatal(got)
	}
}
func TestPRGenerationContextLimits(t *testing.T) {
	client := &http.Client{Transport: generationTransport(func(r *http.Request) (*http.Response, error) {
		var payload struct{ Messages []struct{ Content string } }
		if e := json.NewDecoder(r.Body).Decode(&payload); e != nil {
			t.Fatal(e)
		}
		text := payload.Messages[1].Content
		for _, v := range []struct {
			char string
			n    int
		}{{"t", 8000}, {"c", 20000}, {"s", 20000}, {"p", 60000}} {
			if !strings.Contains(text, strings.Repeat(v.char, v.n)+"\n\n[truncated]") || strings.Contains(text, strings.Repeat(v.char, v.n+1)) {
				t.Fatal("wrong context bound", v)
			}
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"choices":[{"message":{"content":"{\"title\":\"Bounded\",\"body\":\"\"}"}}]}`)), Header: make(http.Header)}, nil
	})}
	_, e := generatePRText(context.Background(), PRGenerationInput{Template: strings.Repeat("t", 8001), CommitSummary: strings.Repeat("c", 20001), DiffSummary: strings.Repeat("s", 20001), DiffPatch: strings.Repeat("p", 60001)}, "fixture", "https://example.invalid", client)
	if e != nil {
		t.Fatal(e)
	}
}
