package api

import "testing"

func TestIMMethodsEmptyInCurrentCodegen(t *testing.T) {
	methods := IMMethods()
	if len(methods) != 0 {
		t.Fatalf("expected no im/* in registry, got %v", methods)
	}
	if got := PreferredIMListMethod(); got != "" {
		t.Fatalf("PreferredIMListMethod = %q, want empty", got)
	}
	if got := PreferredIMHistoryMethod(); got != "" {
		t.Fatalf("PreferredIMHistoryMethod = %q, want empty", got)
	}
	if got := PreferredIMSendMethod(); got != "" {
		t.Fatalf("PreferredIMSendMethod = %q, want empty", got)
	}
}

func TestIMMethodCandidatesDocumented(t *testing.T) {
	if got := ResolveIMListMethod(); got != "im/list" {
		t.Fatalf("ResolveIMListMethod = %q", got)
	}
	if got := ResolveIMHistoryMethod(); got != "im/history" {
		t.Fatalf("ResolveIMHistoryMethod = %q", got)
	}
	if got := ResolveIMSendMethod(); got != "im/send" {
		t.Fatalf("ResolveIMSendMethod = %q", got)
	}
	if len(IMListMethodCandidates()) == 0 || len(IMHistoryMethodCandidates()) == 0 || len(IMSendMethodCandidates()) == 0 {
		t.Fatal("expected non-empty IM method candidates")
	}
	if len(IMCordisChannels()) == 0 {
		t.Fatal("expected Cordis channel prefixes")
	}
}

func TestIsDshIMInventoryEntry(t *testing.T) {
	cases := []struct {
		mod, id string
		want    bool
	}{
		{"@xmanrui/dsh-im", "xmanrui-dsh-im", true},
		{"dsh-im", "anything", true},
		{"@deepseek-ai/other", "xmanrui-dsh-im", true},
		{"@deepseek-ai/dsh-usage-panel", "usage", false},
	}
	for _, tc := range cases {
		if got := IsDshIMInventoryEntry(tc.mod, tc.id); got != tc.want {
			t.Fatalf("IsDshIMInventoryEntry(%q,%q)=%v want %v", tc.mod, tc.id, got, tc.want)
		}
	}
}
