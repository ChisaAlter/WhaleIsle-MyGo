package settings

import "testing"

func TestSectionForNamespaceKnown(t *testing.T) {
	tests := []struct {
		ns   string
		want SectionID
	}{
		{"account", SectionAccount},
		{"general", SectionGeneral},
		{"appearance", SectionAppearance},
		{"theme", SectionAppearance},
		{"interface", SectionInterface},
		{"ui", SectionInterface},
		{"agentPresets", SectionAgentPresets},
		{"agent-presets", SectionAgentPresets},
		{"usage-stats", SectionUsageStats},
		{"remoteWorkspace", SectionRemoteWorkspace},
		{"permission", SectionOther},
		{"", SectionOther},
		{"unknown-plugin-ns", SectionOther},
	}
	for _, tc := range tests {
		if got := SectionForNamespace(tc.ns); got != tc.want {
			t.Errorf("SectionForNamespace(%q) = %q, want %q", tc.ns, got, tc.want)
		}
	}
}

func TestGroupNamespacesOrderAndBuckets(t *testing.T) {
	groups := GroupNamespaces([]string{"permission", "account", "general", "usageStats", "zzz"})
	if len(groups) != 4 {
		t.Fatalf("groups = %d, want 4 (account, usage-stats, general, other)", len(groups))
	}
	if groups[0].Section != SectionAccount || len(groups[0].Items) != 1 || groups[0].Items[0] != "account" {
		t.Fatalf("first group = %+v", groups[0])
	}
	if groups[1].Section != SectionUsageStats || groups[1].Items[0] != "usageStats" {
		t.Fatalf("usage group = %+v", groups[1])
	}
	if groups[2].Section != SectionGeneral {
		t.Fatalf("third group = %+v", groups[2])
	}
	if groups[3].Section != SectionOther {
		t.Fatalf("fourth group = %+v, want other with permission+zzz", groups[3])
	}
	other := groups[3].Items
	if len(other) != 2 || other[0] != "permission" || other[1] != "zzz" {
		t.Fatalf("other items = %v", other)
	}
}
