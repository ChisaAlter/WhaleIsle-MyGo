package composer

import "testing"

func TestParseDraftMentions(t *testing.T) {
	got := ParseDraftMentions("see @src/a.go and /code-review please")
	if len(got) != 2 {
		t.Fatalf("len=%d %+v", len(got), got)
	}
	if got[0].Kind != DraftMentionAt || got[0].Text != "@src/a.go" {
		t.Fatalf("at: %+v", got[0])
	}
	if got[1].Kind != DraftMentionSkill || got[1].Text != "/code-review" {
		t.Fatalf("skill: %+v", got[1])
	}
}

func TestParseDraftMentionsSkipsBareAndMidToken(t *testing.T) {
	if got := ParseDraftMentions("@ /"); len(got) != 0 {
		t.Fatalf("bare: %+v", got)
	}
	if got := ParseDraftMentions("https://example.com foo/bar"); len(got) != 0 {
		t.Fatalf("mid-token: %+v", got)
	}
	got := ParseDraftMentions("/skill\n@dir/sub ")
	if len(got) != 2 || got[0].Text != "/skill" || got[1].Text != "@dir/sub" {
		t.Fatalf("%+v", got)
	}
}

func TestRemoveDraftMention(t *testing.T) {
	draft := "hi @src/a.go /review more"
	tags := ParseDraftMentions(draft)
	if len(tags) != 2 {
		t.Fatalf("%+v", tags)
	}
	draft = RemoveDraftMention(draft, tags[0])
	if draft != "hi /review more" {
		t.Fatalf("after at: %q", draft)
	}
	tags = ParseDraftMentions(draft)
	draft = RemoveDraftMention(draft, tags[0])
	if draft != "hi more" {
		t.Fatalf("after skill: %q", draft)
	}
}

func TestReorderDraftMentions(t *testing.T) {
	draft := "see @a.go /review @b.go end"
	// Move last to front: others shift right.
	got := ReorderDraftMentions(draft, 2, 0)
	if got != "see @b.go @a.go /review end" {
		t.Fatalf("move last to front: %q", got)
	}
	// Drop first onto last (insert before last after removal).
	got = ReorderDraftMentions(draft, 0, 2)
	if got != "see /review @a.go @b.go end" {
		t.Fatalf("move first before last: %q", got)
	}
	if ReorderDraftMentions(draft, 1, 1) != draft {
		t.Fatalf("noop same index")
	}
	if ReorderDraftMentions(draft, 0, 1) != draft {
		t.Fatalf("noop adjacent insert-after-self")
	}
	if ReorderDraftMentions(draft, -1, 0) != draft || ReorderDraftMentions(draft, 0, 9) != draft {
		t.Fatalf("out of range should noop")
	}
}

func TestReplaceDraftMention(t *testing.T) {
	draft := "hi @old.go /skill more"
	tags := ParseDraftMentions(draft)
	got := ReplaceDraftMention(draft, tags[0], "@new.go")
	if got != "hi @new.go /skill more" {
		t.Fatalf("replace at: %q", got)
	}
	tags = ParseDraftMentions(got)
	got = ReplaceDraftMention(got, tags[1], "/other")
	if got != "hi @new.go /other more" {
		t.Fatalf("replace skill: %q", got)
	}
	tags = ParseDraftMentions(got)
	got = ReplaceDraftMention(got, tags[0], "")
	if got != "hi /other more" {
		t.Fatalf("empty removes: %q", got)
	}
}

func TestMentionRuneRange(t *testing.T) {
	draft := "看 @文件.go /技能"
	tags := ParseDraftMentions(draft)
	if len(tags) != 2 {
		t.Fatalf("%+v", tags)
	}
	start, end, ok := MentionRuneRange(draft, tags[0])
	if !ok || start != 2 || end != 2+len([]rune(tags[0].Text)) {
		t.Fatalf("at range %d:%d ok=%v text=%q", start, end, ok, tags[0].Text)
	}
	start, end, ok = MentionRuneRange(draft, tags[1])
	if !ok {
		t.Fatal("skill range missing")
	}
	runes := []rune(draft)
	if string(runes[start:end]) != tags[1].Text {
		t.Fatalf("skill slice %q want %q", string(runes[start:end]), tags[1].Text)
	}
}

func TestChipLabel(t *testing.T) {
	if got := ChipLabel(DraftMention{Kind: DraftMentionAt, Text: "@src/a.go"}); got != "file @src/a.go" {
		t.Fatalf("at: %q", got)
	}
	if got := ChipLabel(DraftMention{Kind: DraftMentionSkill, Text: "/review"}); got != "skill /review" {
		t.Fatalf("skill: %q", got)
	}
}
