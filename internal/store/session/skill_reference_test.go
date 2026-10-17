package session

import (
	"testing"
	"whaleisle/internal/dsh/api"
)

func TestComposerSkillReferencePathsSurviveBothCatalogs(t *testing.T) {
	path := "/skills/review/SKILL.md"
	list := flattenSkillList(api.SkillListValue{Skills: []api.SkillListValueSkillsItem{{Name: "review", Path: &path}}})
	inventory := flattenSkillInventory(api.SkillInventorySnapshot{Skills: []api.SkillInventorySnapshotSkillsItem{{Name: "review", Path: &path, UserInvocable: true}}})
	for _, rows := range [][]SkillOption{list, inventory} {
		if len(rows) != 1 || rows[0].Path != path {
			t.Fatalf("preview path lost: %+v", rows)
		}
	}
}
