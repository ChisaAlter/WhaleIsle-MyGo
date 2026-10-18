package composer

import (
	"testing"

	"whaleisle/internal/store/session"
)

func TestSkillInsert(t *testing.T) {
	if got := SkillInsert("my-skill"); got != "/my-skill " {
		t.Fatalf("got %q", got)
	}
}

func TestAppendSkillInvocation(t *testing.T) {
	draft := "hello"
	AppendSkillInvocation(&draft, "code-review")
	if draft != "hello /code-review " {
		t.Fatalf("got %q", draft)
	}
	draft = ""
	AppendSkillInvocation(&draft, "x")
	if draft != "/x " {
		t.Fatalf("empty got %q", draft)
	}
}

func TestFilterSkillsSubstring(t *testing.T) {
	all := []session.SkillOption{
		{Name: "Code-Review"},
		{Name: "summarize"},
		{Name: "git-commit"},
	}
	got := FilterSkills(all, "VIEW", 10)
	if len(got) != 1 || got[0].Name != "Code-Review" {
		t.Fatalf("%+v", got)
	}
	got = FilterSkills(all, "git", 10)
	if len(got) != 1 || got[0].Name != "git-commit" {
		t.Fatalf("%+v", got)
	}
	got = FilterSkills(all, "", 2)
	if len(got) != 2 {
		t.Fatalf("limit: %+v", got)
	}
}
