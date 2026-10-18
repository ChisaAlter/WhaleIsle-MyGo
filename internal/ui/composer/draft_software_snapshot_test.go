package composer

import (
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// These exercise the shipped stable DraftClient path through MyGo's software
// renderer. They are evidence of layout/ink, not a live Windows parity claim.
func TestShippedDraftSoftwareSnapshots(t *testing.T) {
	dir := os.Getenv("CHAT_COMPOSER_SNAPSHOT_DIR")
	if dir == "" {
		t.Skip("set CHAT_COMPOSER_SNAPSHOT_DIR to export software renders")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, width := range []int{320, 800} {
		for _, hero := range []bool{false, true} {
			for _, dark := range []bool{false, true} {
				for _, variant := range []string{"placeholder", "selected", "scroll"} {
					draft := ""
					if variant == "selected" {
						draft = "A😀B\n中文 selected text\nReady for the next step"
					} else if variant == "scroll" {
						for line := 1; line <= 20; line++ {
							draft += fmt.Sprintf("Draft line %02d\n", line)
						}
						draft = strings.TrimSuffix(draft, "\n")
					}
					client := NewDraftClient(&draft)
					model := "DeepSeek-V3.2-Exp (reasoning enabled)"
					tt := ui.NewTester(func(c *ui.Context) {
						if dark {
							c.SetTheme(ui.DarkTheme())
						}
						colors := theme.Default().Active(dark)
						ui.Column(c).Fill().Background(chrome.Color(colors.BG)).Padding(16, 0, 0, 0).Children(func() {
							View(c, Model{Draft: &draft, TextInput: client, UseTextInputClient: true, Hero: hero, SystemDark: dark, WorkspaceRoot: "fixture", ModelPick: &model})
						})
					}, width, 650)
					if variant == "selected" {
						if err := tt.Click("Composer draft"); err != nil {
							t.Fatal(err)
						}
						client.SetSelectionUTF16(0, 4)
						tt.Frame()
					} else if variant == "scroll" {
						client.SetSelectionUTF16(0, 0)
						tt.Frame()
						input, _ := tt.Find("Composer draft")
						tt.Scroll(input.X+30, input.Y+30, 0, 48)
					}
					name := fmt.Sprintf("draft-client-%d-hero-%v-dark-%v-%s", width, hero, dark, variant)
					f, err := os.Create(filepath.Join(dir, name+".png"))
					if err != nil {
						t.Fatal(err)
					}
					if err := png.Encode(f, tt.Image()); err != nil {
						t.Fatal(err)
					}
					if err := f.Close(); err != nil {
						t.Fatal(err)
					}
					geometry := map[string]any{"nativeClient": true, "stableDraftOwner": client.Draft == &draft, "fontSize": client.font.Size, "lineHeight": draftInputLineHeight, "wrapWidth": client.wrapWidth, "scrollY": client.scrollY, "selection": client.selection.Range}
					for _, label := range []string{"Composer card", "Composer draft", "Composer placeholder", i18n.T(i18n.KeyComposerSend)} {
						if r, ok := tt.Find(label); ok {
							geometry[label] = r
						}
					}
					data, err := json.MarshalIndent(geometry, "", "  ")
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(dir, name+".geometry.json"), data, 0644); err != nil {
						t.Fatal(err)
					}
				}
			}
		}
	}
}
