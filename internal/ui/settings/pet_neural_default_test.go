package settings

import (
	"os"
	"path/filepath"
	"testing"
	"whaleisle/internal/config"
	"whaleisle/internal/desktop/pet"
)

func TestPetNeuralPreferenceSurvivesAppearanceAutosave(t *testing.T) {
	for _, tc := range []struct {
		name, raw string
		want      bool
	}{
		{name: "fresh absent namespace", want: true},
		{name: "fresh empty document", raw: `{}`, want: true},
		{name: "existing explicit false", raw: `{"mygoDesktopPet":{"neural":false}}`},
		{name: "existing omitted false", raw: `{"mygoDesktopPet":{}}`},
		{name: "existing null false", raw: `{"mygoDesktopPet":{"neural":null}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.raw != "" {
				if err := os.WriteFile(filepath.Join(dir, config.FileName), []byte(tc.raw), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			store, err := config.Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			before := store.DesktopPet()
			if before.Neural != tc.want {
				t.Fatal("fixture preference changed before first save")
			}
			before.Behavior.Scale = 1.4
			// A different pending renderer/skin draft must stay out of the
			// immediate appearance write. Readiness stays false: no GUI.
			p := &pet.Companion{Store: store, Neural: !tc.want, SkinID: "unsubmitted-draft", Behavior: before.Behavior}
			savePetSetting(p, nil, "scale")
			if p.Error != "" {
				t.Fatal(p.Error)
			}
			reopened, err := config.Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			if got := reopened.DesktopPet(); got.Neural != tc.want || got.Enabled || got.SkinID != "" || got.Behavior.Scale != 1.4 {
				t.Fatalf("autosave changed renderer/window/draft: %+v", got)
			}
			if p.Neural == tc.want || p.SkinID != "unsubmitted-draft" {
				t.Fatal("appearance autosave erased a pending draft")
			}
		})
	}
}
