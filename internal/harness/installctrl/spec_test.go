package installctrl

import "testing"

func TestIsValidGithubSpec(t *testing.T) {
	ok := []string{
		"github:owner/repo",
		"github:DeepSeek-AI/dsh-im",
		"github:owner/repo#main",
	}
	for _, spec := range ok {
		if !IsValidGithubSpec(spec) {
			t.Fatalf("want valid: %q", spec)
		}
	}
	bad := []string{"", "owner/repo", "github:", "github:bad!/repo", "npm:foo"}
	for _, spec := range bad {
		if IsValidGithubSpec(spec) {
			t.Fatalf("want invalid: %q", spec)
		}
	}
}

func TestNormalizeAllowBuilds(t *testing.T) {
	if got := NormalizeAllowBuilds(nil); got == nil || len(got) != 0 {
		t.Fatalf("nil -> empty slice, got %v", got)
	}
	if got := NormalizeAllowBuilds([]string{"esbuild", "esbuild"}); len(got) != 1 {
		t.Fatalf("dedupe: %v", got)
	}
	if got := NormalizeAllowBuilds([]string{"../evil"}); got != nil {
		t.Fatalf("invalid: %v", got)
	}
}
