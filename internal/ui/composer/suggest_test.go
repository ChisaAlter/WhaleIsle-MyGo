package composer

import "testing"

func TestSlashToken(t *testing.T) {
	q, tok, ok := SlashToken("/cle")
	if !ok || q != "cle" || tok.Start != 0 || tok.End != 4 {
		t.Fatalf("got %q %+v %v", q, tok, ok)
	}
	if _, _, ok := SlashToken("hello"); ok {
		t.Fatal("expected no slash")
	}
	if _, _, ok := SlashToken("/a\nb"); ok {
		t.Fatal("expected no slash across newline")
	}
}

func TestMentionToken(t *testing.T) {
	q, tok, ok := MentionToken("see @src/ma")
	if !ok || q != "src/ma" || tok.Start != 4 || tok.End != len("see @src/ma") {
		t.Fatalf("got %q %+v %v", q, tok, ok)
	}
	q, tok, ok = MentionToken("@")
	if !ok || q != "" || tok.Start != 0 {
		t.Fatalf("empty query: %q %+v", q, tok)
	}
}

func TestActiveSuggestPrefersSlash(t *testing.T) {
	k, q, _, ok := ActiveSuggest("/new")
	if !ok || k != SuggestSlash || q != "new" {
		t.Fatalf("got %v %q", k, q)
	}
}

func TestFilterSlashCommands(t *testing.T) {
	all := MergeSlashCommands(StubSlashCommands(), []SlashCommand{
		{Line: "/compact", Description: "Compact"},
	})
	got := FilterSlashCommands(all, "cl", 10)
	if len(got) != 1 || got[0].Line != "/clear" {
		t.Fatalf("%+v", got)
	}
}

func TestFilterPaths(t *testing.T) {
	paths := []string{"README.md", "internal/ui/composer/suggest.go", "cmd/whaleisle/main.go"}
	got := FilterPaths(paths, "comp", 5)
	if len(got) != 1 || got[0] != "internal/ui/composer/suggest.go" {
		t.Fatalf("%+v", got)
	}
}

func TestReplaceToken(t *testing.T) {
	d := ReplaceToken("hi @src", TokenRange{3, 7}, "@internal/foo.go")
	if d != "hi @internal/foo.go" {
		t.Fatalf("%q", d)
	}
}

func TestLocalSlashAction(t *testing.T) {
	if LocalSlashAction("/new", StubSlashCommands()) != "new" {
		t.Fatal()
	}
	if LocalSlashAction("/compact", StubSlashCommands()) != "" {
		t.Fatal()
	}
}
