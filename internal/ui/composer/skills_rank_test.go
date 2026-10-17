package composer

import (
	"fmt"
	"testing"
	"whaleisle/internal/store/session"
)

func TestSkillsOrderedSubsequenceRanking(t *testing.T) {
	all := []session.SkillOption{{Name: "code-review", Path: "keep"}, {Name: "cr-helper"}, {Name: "create"}, {Name: "unrelated"}}
	got := FilterSkills(all, "CR", 2)
	if len(got) != 2 || got[0].Name != "create" && got[0].Name != "cr-helper" {
		t.Fatalf("prefix ranking: %+v", got)
	}
	got = FilterSkills(all, "cdrv", 10)
	if len(got) != 1 || got[0].Name != "code-review" || got[0].Path != "keep" {
		t.Fatalf("ordered gaps/metadata: %+v", got)
	}
	if len(FilterSkills(all, "vrc", 10)) != 0 {
		t.Fatal("unordered letters matched")
	}
}

func TestSkillsRankingStableAndUncappedCatalog(t *testing.T) {
	all := make([]session.SkillOption, 30)
	for i := range all {
		all[i] = session.SkillOption{Name: "same", Description: fmt.Sprint(i)}
	}
	got := FilterSkills(all, "", len(all))
	if len(got) != 30 || got[29].Description != "29" {
		t.Fatal("catalog clipped or reordered")
	}
	got = FilterSkills(all, "sm", len(all))
	if len(got) != 30 || got[0].Description != "0" || got[29].Description != "29" {
		t.Fatal("equal scores lost stable order")
	}
}
