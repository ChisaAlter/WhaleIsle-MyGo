package chat

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Software-only evidence: this neither launches Windows nor compares live
// Electron pixels, system fonts, native GPU output, or backend behavior.
func TestChatComposerSoftwareSnapshots(t *testing.T) {
	dir := os.Getenv("CHAT_COMPOSER_SNAPSHOT_DIR")
	if dir == "" {
		t.Skip("set CHAT_COMPOSER_SNAPSHOT_DIR to export software renders")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(dir, "attachment-fixture.png")
	f, err := os.Create(fixture)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 160, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 160; x++ {
			img.SetRGBA(x, y, color.RGBA{31, 80, 96, 255})
		}
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	for _, width := range []int{320, 400, 800, 1200} {
		for _, dark := range []bool{false, true} {
			for _, attachments := range []bool{false, true} {
				draft := "A multiline draft\nReady for the next step"
				client := composer.NewDraftClient(&draft)
				model := "DeepSeek-V3.2-Exp (reasoning enabled)"
				msgs := []session.Message{presentationMessage("A short user message"), {ID: "assistant-1", Role: "assistant", Kind: session.MessageKindText, Body: "Assistant **Markdown** stays formatted, with `inline code`."}, presentationMessage("```diff\n-old\n+new\n```\nThe user source remains literal.")}
				if attachments {
					msgs = []session.Message{presentationMessage("[image] " + fixture)}
				}
				tt := ui.NewTester(func(c *ui.Context) {
					if dark {
						c.SetTheme(ui.DarkTheme())
					}
					colors := theme.Default().Active(dark)
					ui.Column(c).Fill().Background(chrome.Color(colors.BG)).Children(func() {
						ui.Box(c).Grow(1).MinHeight(0).Children(func() {
							View(c, Model{Snapshot: session.ViewState{Messages: msgs}, SystemDark: dark, ContentWidth: 748, OnQuoteMessage: func(string) {}, OnEditMessage: func(float64, string) {}, OnOpenImageLightbox: func(string) {}})
						})
						composer.View(c, composer.Model{Draft: &draft, UseTextInputClient: true, TextInput: client, WorkspaceRoot: "fixture", SystemDark: dark, ModelPick: &model, Snapshot: session.ViewState{ModelLabels: []string{model, "small"}}, OnPromptModeChange: func(session.PromptMode) {}, OnToggleSkills: func(bool) {}, OnSend: func(string) {}})
					})
				}, width, 650)
				name := fmt.Sprintf("chat-composer-%d-dark-%v-attachments-%v", width, dark, attachments)
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
				boxes := map[string]ui.Rect{}
				for _, label := range []string{"User message bubble", "User message actions", "Composer card", "Composer draft", i18n.T(i18n.KeyComposerSend)} {
					if r, ok := tt.Find(label); ok {
						boxes[label] = r
					}
				}
				data, err := json.MarshalIndent(boxes, "", "  ")
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
