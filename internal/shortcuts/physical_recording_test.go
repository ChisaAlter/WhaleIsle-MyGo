package shortcuts

import (
	"errors"
	"sort"
	"strings"
	"testing"
)

func TestDesktopPhysicalGrammarPreservesExactModifiersAndPairs(t *testing.T) {
	for _, platform := range []string{"windows", "darwin"} {
		for raw, want := range map[string]string{"Meta+Control+Shift+Alt+F24": "Control+Alt+Shift+Meta+F24", "Control+B & A": "Control+A & B", "F13": "F13", "Space": "Space", "Alt+F4": "Alt+F4"} {
			got, err := parseForPlatform(raw, platform)
			if err != nil || got.String() != want {
				t.Fatal(platform, raw, got, err)
			}
			binding := got.PhysicalBinding()
			restored := SharedDisplay(binding, map[string]string{"windows": "windows", "darwin": "macos"}[platform])
			round, err := parseForPlatform(restored, platform)
			if err != nil || binding.Code != round.PhysicalBinding().Code || binding.SecondCode != round.PhysicalBinding().SecondCode || physicalModifierSet(binding.Modifiers, platform) != physicalModifierSet(round.PhysicalBinding().Modifiers, platform) {
				t.Fatal("physical roundtrip lost modifiers or second code", raw, restored, err)
			}
		}
		for _, raw := range []string{"Control+A & A", "Control+A & B & C", "Alt+Alt+F9", "Control+Unknown"} {
			if _, err := parseForPlatform(raw, platform); err == nil {
				t.Fatal("invalid physical grammar accepted", platform, raw)
			}
		}
	}
	for _, raw := range []string{"F13", "Alt+F4", "Mod+A & B", "Mod+S"} {
		if _, err := parseForPlatform(raw, "linux"); err == nil {
			t.Fatal("legacy platform grammar broadened", raw)
		}
	}
}

func TestRecordedInlineSaveRetainsCandidateThroughReviewAndNarrowRetry(t *testing.T) {
	calls := []map[string]string{}
	stale := true
	e := NewEditor(nil, nil, func(p map[string]string) error {
		copy := map[string]string{}
		for k, v := range p {
			copy[k] = v
		}
		calls = append(calls, copy)
		if stale {
			return errors.New("stale revision")
		}
		return nil
	})
	e.UseShared(nil)
	*e.Draft("session.search") = "Mod+Alt+F11"
	e.BeginRecording("window.refresh")
	if e.RecordAndSave("Mod+Shift+F9") || e.Err == "" || e.Recording != "window.refresh" {
		t.Fatal("stale save lost recording state", e.Err)
	}
	if e.Active()["window.refresh"] != "Mod+F5" || *e.Draft("window.refresh") != "Mod+Shift+F9" {
		t.Fatal("failed write changed accepted binding or lost candidate")
	}
	e.SetReload(func() (map[string]string, error) {
		values := e.Active()
		values["window.refresh"] = "Mod+F8"
		return values, nil
	})
	if !e.Review() || len(calls) != 1 || *e.Draft("window.refresh") != "Mod+Shift+F9" {
		t.Fatal("review saved or discarded the candidate")
	}
	stale = false
	if !e.SaveRow("window.refresh", false) || len(calls) != 2 {
		t.Fatal("explicit retry failed", e.Err)
	}
	for _, patch := range calls {
		if len(patch) != 1 || patch["window.refresh"] != "Mod+Shift+F9" {
			t.Fatal("row save included another draft", patch)
		}
	}
	if e.Active()["session.search"] == "Mod+Alt+F11" || *e.Draft("session.search") != "Mod+Alt+F11" {
		t.Fatal("narrow save committed or erased unrelated draft")
	}
	e.SetDescribe(func(id, raw string) string { return "conflicts with product owner" })
	e.BeginRecording("window.refresh")
	if e.RecordAndSave("Mod+Shift+F10") || len(calls) != 2 || e.Err == "" {
		t.Fatal("cross-owner conflict reached persistence")
	}
}

// Primary is a platform alias; all other explicit modifiers retain their owner.
func physicalModifierSet(modifiers []string, platform string) string {
	out := append([]string(nil), modifiers...)
	for i, m := range out {
		if m == "primary" {
			out[i] = "control"
			if platform == "darwin" {
				out[i] = "meta"
			}
		}
	}
	sort.Strings(out)
	return strings.Join(out, "+")
}
