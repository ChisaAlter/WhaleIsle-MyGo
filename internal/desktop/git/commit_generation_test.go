package git

import (
	"context"
	"io"
	"net/http"
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
