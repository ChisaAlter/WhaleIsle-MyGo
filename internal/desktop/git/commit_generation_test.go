package git

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

type generationTransport func(*http.Request) (*http.Response, error)

func (f generationTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestCommitGenerationOriginalHeuristic(t *testing.T) {
	g, e := generateCommitText(context.Background(), CommitGenerationInput{Summary: "A\tnew.txt\nM\told.txt"}, "", "", nil)
	if e != nil || g.Subject != "Add new.txt and 1 other files" || g.Source != "heuristic" {
		t.Fatalf("%+v %v", g, e)
	}
}
func TestCommitGenerationOriginalRequestAndSanitize(t *testing.T) {
	client := &http.Client{Transport: generationTransport(func(r *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(r.Body)
		if r.URL.String() != "https://example.invalid/v1/chat/completions" || !strings.Contains(string(b), `"model":"deepseek-chat"`) || !strings.Contains(string(b), "Recent commit subjects") || !strings.Contains(string(b), "Staged patch") {
			t.Fatalf("wrong original contract: %s %s", r.URL, b)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"choices":[{"message":{"content":"{\"subject\":\"Add selected files.\",\"body\":\"Details\",\"branch\":\"feature/selected\"}"}}]}`)), Header: make(http.Header)}, nil
	})}
	g, e := generateCommitText(context.Background(), CommitGenerationInput{Summary: "A\ta", Patch: "+a", RecentSubjects: "previous", IncludeBranch: true}, "fixture-only", "https://example.invalid", client)
	if e != nil || g.Subject != "Add selected files" || g.Body != "Details" || g.Source != "model" {
		t.Fatalf("%+v %v", g, e)
	}
}
func TestCommitGenerationConfiguredFailureAndCancellation(t *testing.T) {
	for _, body := range []string{`{}`, `{"choices":[{"message":{"content":"not json"}}]}`} {
		client := &http.Client{Transport: generationTransport(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		})}
		if _, e := generateCommitText(context.Background(), CommitGenerationInput{Summary: "A\ta"}, "fixture", "https://example.invalid", client); e == nil {
			t.Fatal("model failure silently fell back")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := generateCommitText(ctx, CommitGenerationInput{Summary: "A\ta"}, "", "", nil); e == nil {
		t.Fatal("cancel ignored")
	}
	if _, e := generateCommitText(context.Background(), CommitGenerationInput{}, "fixture", "http://example.invalid", nil); e == nil {
		t.Fatal("cleartext credentials allowed")
	}
}

// TestMergeGenerationSummaryNamesUntrackedFiles pins the title-bar input: the
// tracked name-status rows stay as-is and the untracked files the review flow
// would commit are added once, so the heuristic can name them.
func TestMergeGenerationSummaryNamesUntrackedFiles(t *testing.T) {
	changes := []Change{
		{Index: 'M', Worktree: ' ', Path: "tracked.txt"},
		{Index: '?', Worktree: '?', Path: "new.txt"},
		{Index: '?', Worktree: '?', Path: "new.txt"},
	}
	got := mergeGenerationSummary("M\ttracked.txt", changes, true)
	if got != "M\ttracked.txt\n??\tnew.txt" {
		t.Fatalf("summary = %q", got)
	}
	// Without a usable HEAD diff every change is reported, staged or not.
	got = mergeGenerationSummary("", []Change{{Index: 'A', Worktree: ' ', Path: "staged.txt"}}, false)
	if got != "A\tstaged.txt" {
		t.Fatalf("fallback summary = %q", got)
	}
	if got := mergeGenerationSummary("", nil, true); got != "" {
		t.Fatalf("empty summary = %q", got)
	}
}

// TestGenerateWorktreeCommitMessageFailsWithoutChanges keeps a clean tree from
// producing invented copy. The directory lives outside this repository, because
// Git reports the enclosing repository's changes from any nested path.
func TestGenerateWorktreeCommitMessageFailsWithoutChanges(t *testing.T) {
	dir, err := os.MkdirTemp("", "whaleisle-generation-*")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer os.RemoveAll(dir)
	if _, err := GenerateWorktreeCommitMessage(context.Background(), dir); err == nil {
		t.Fatal("clean directory generated a message")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := GenerateWorktreeCommitMessage(ctx, dir); err == nil {
		t.Fatal("cancelled generation continued")
	}
}
