package composer

import (
	"fmt"
	"testing"
	"whaleisle/internal/store/session"
)

func TestReferenceCanonicalIdentity(t *testing.T) {
	one, two := `@[same title](dsh-session:ImEi)`, `@[same title](dsh-session:ImIi)`
	m := Model{OnSearchReferences: func(q string, quoted bool) []session.ReferenceCandidate {
		return []session.ReferenceCandidate{
			{Kind: "session", Label: "same title", SessionID: "a", Mention: one}, {Kind: "session", Label: "same title", SessionID: "b", Mention: two},
			{Kind: "file", Path: "my file.txt"}, {Kind: "directory", Path: "my folder"},
		}
	}}
	got := buildSuggestItems(m, SuggestMention, "")
	if len(got) != 4 || got[0].insert != one+" " || got[1].insert != two+" " || got[0].sub == got[1].sub || got[2].insert != `@"my file.txt" ` || got[3].insert != `@"my folder/` {
		t.Fatalf("%+v", got)
	}
	tags := ParseDraftMentions("read " + one + ` and @"my file.txt" now`)
	if len(tags) != 2 || tags[0].Text != one || tags[0].Kind != DraftMentionSession || tags[1].Text != `@"my file.txt"` {
		t.Fatalf("%+v", tags)
	}
	if ChipLabel(tags[0]) != "session @same title" {
		t.Fatal(ChipLabel(tags[0]))
	}
	if RemoveDraftMention("read "+one+` and @"my file.txt" now`, tags[1]) != "read "+one+" and now" {
		t.Fatal("partial quoted removal")
	}
}

// A scrollable candidate list must not discard session rows after file hits.
func TestReferenceCandidatesRemainReachableAfterFiles(t *testing.T) {
	candidates := make([]session.ReferenceCandidate, 0, 10)
	for i := 0; i < 9; i++ {
		candidates = append(candidates, session.ReferenceCandidate{Kind: "file", Path: fmt.Sprintf("file-%d.txt", i)})
	}
	mention := "@[same title](dsh-session:ImEi)"
	candidates = append(candidates, session.ReferenceCandidate{Kind: "session", Label: "same title", SessionID: "a", Mention: mention})
	m := Model{OnSearchReferences: func(string, bool) []session.ReferenceCandidate { return candidates }}
	got := buildSuggestItems(m, SuggestMention, "")
	if len(got) != 10 || got[9].insert != mention+" " {
		t.Fatalf("session candidate truncated: %+v", got)
	}
}
