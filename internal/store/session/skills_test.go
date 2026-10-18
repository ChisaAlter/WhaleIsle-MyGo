package session

import (
	"testing"

	"whaleisle/internal/dsh/api"
)

func TestFlattenSkillList(t *testing.T) {
	when := "when debugging"
	list := api.SkillListValue{
		Skills: []api.SkillListValueSkillsItem{
			{Name: "alpha", Description: "A skill", ModelInvocable: true},
			{Name: "", Description: "skip"},
			{Name: "user-only", Description: "", WhenToUse: &when, ModelInvocable: false},
		},
	}
	opts := flattenSkillList(list)
	if len(opts) != 2 {
		t.Fatalf("len = %d want 2", len(opts))
	}
	if opts[0].Name != "alpha" || opts[0].UserOnly {
		t.Fatalf("first = %+v", opts[0])
	}
	if opts[1].Name != "user-only" || !opts[1].UserOnly || opts[1].Description != when {
		t.Fatalf("second = %+v", opts[1])
	}
}
