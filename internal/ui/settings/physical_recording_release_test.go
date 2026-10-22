package settings

import (
	"github.com/egoist/mygo/ui"
	"runtime"
	"testing"
	"whaleisle/internal/shortcuts"
)

func recorderFixture(t *testing.T) (*ui.Tester, *shortcuts.Editor, *[]map[string]string) {
	t.Helper()
	writes := []map[string]string{}
	e := shortcuts.NewEditor(nil, nil, func(patch map[string]string) error {
		copy := map[string]string{}
		for k, v := range patch {
			copy[k] = v
		}
		writes = append(writes, copy)
		return nil
	})
	e.BeginRecording("window.refresh")
	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { shortcutRecorder(c, e, "window.refresh", nil) }) }, 580, 280)
	return tt, e, &writes
}

func TestPhysicalRecordingWaitsForReleaseAndSavesOnlyOneRow(t *testing.T) {
	tt, e, writes := recorderFixture(t)
	*e.Draft("session.search") = "Mod+Alt+F11"
	tt.PhysicalKeyDown("F9", ui.KeyF9, ui.Cmd, false, false, false)
	tt.PhysicalKeyDown("F9", ui.KeyF9, ui.Cmd, true, false, false)
	if len(*writes) != 0 || e.Recording == "" {
		t.Fatal("keydown/repeat saved before release")
	}
	tt.PhysicalKeyDown("F10", ui.KeyF10, ui.Cmd, false, false, false)
	if len(*writes) != 0 {
		t.Fatal("simultaneous second key saved before release")
	}
	tt.PhysicalKeyUp("F10", ui.KeyF10, ui.Cmd, false)
	if len(*writes) != 1 {
		t.Fatal("release did not autosave exactly once", *writes, e.Err)
	}
	patch := (*writes)[0]
	if len(patch) != 1 || patch["window.refresh"] == "" || *e.Draft("session.search") != "Mod+Alt+F11" || e.Active()["session.search"] == "Mod+Alt+F11" {
		t.Fatal("recording saved an unrelated draft", patch)
	}
	chord, err := shortcuts.Parse(patch["window.refresh"])
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		if chord.PhysicalBinding().Code != "F10" || chord.PhysicalBinding().SecondCode != "F9" {
			t.Fatal("two physical keys lost", chord)
		}
	} else if chord.Key != "F10" || chord.SecondKey != "" {
		t.Fatal("legacy platform did not retain its single-key grammar", chord)
	}
	tt.PhysicalKeyUp("F9", ui.KeyF9, ui.Cmd, false)
	if len(*writes) != 1 {
		t.Fatal("remaining release saved twice")
	}
}

func TestPhysicalRecordingModifierReleaseSavesAndInputGuardsCancelPending(t *testing.T) {
	t.Run("modifier-release", func(t *testing.T) {
		tt, e, writes := recorderFixture(t)
		tt.PhysicalKeyDown("F9", ui.KeyF9, ui.Cmd, false, false, false)
		tt.HoldModifiers(0)
		if len(*writes) != 1 || e.Recording != "" {
			t.Fatal("modifier release did not save pending chord", e.Err, *writes)
		}
	})
	for _, guard := range []string{"compose", "altgr", "dead", "blur", "missing-code", "three-keys"} {
		t.Run(guard, func(t *testing.T) {
			tt, e, writes := recorderFixture(t)
			tt.PhysicalKeyDown("F9", ui.KeyF9, ui.Cmd, false, false, false)
			switch guard {
			case "compose":
				tt.Compose("candidate", 9)
			case "altgr":
				tt.PhysicalKeyDown("KeyQ", ui.KeyQ, ui.Ctrl|ui.Alt, false, true, false)
			case "dead":
				tt.PhysicalKeyDown("Quote", ui.KeyQuote, ui.Cmd, false, false, true)
			case "blur":
				tt.SetFocused(false)
			case "missing-code":
				tt.PhysicalKeyDown("", ui.KeyF10, ui.Cmd, false, false, false)
			case "three-keys":
				tt.PhysicalKeyDown("F10", ui.KeyF10, ui.Cmd, false, false, false)
				tt.PhysicalKeyDown("F11", ui.KeyF11, ui.Cmd, false, false, false)
			}
			tt.PhysicalKeyUp("F9", ui.KeyF9, ui.Cmd, false)
			tt.PhysicalKeyUp("F10", ui.KeyF10, ui.Cmd, false)
			tt.PhysicalKeyUp("F11", ui.KeyF11, ui.Cmd, false)
			if len(*writes) != 0 || e.Recording != "window.refresh" {
				t.Fatal("guarded/incomplete recording escaped to persistence", guard, *writes, e.Err)
			}
		})
	}
}
