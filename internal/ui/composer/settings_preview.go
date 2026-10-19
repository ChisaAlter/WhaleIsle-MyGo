package composer

import (
	"encoding/json"
	"github.com/egoist/mygo/ui"
)

// SettingsEffectPreview uses the shipped native painters and an isolated draft.
// It has no session, submission, attachment, or Host callbacks.
func SettingsEffectPreview(c *ui.Context, style json.RawMessage, typing bool) {
	box := ui.Column(c.Key("settings-effect-preview")).FillWidth().MinWidth(0).Padding(12).Radius(16).Background(c.Theme().Surface)
	text := ui.Local(box, "text", func() string { return "" })
	client := ui.Local(box, "client", func() *DraftClient { return NewDraftClient(text) })
	reducedMotion := c.Preferences().ReduceMotion
	beam := readBeamStyle(style)
	beamState := ui.Local(box, "preview-beam-timeline", func() beamTimeline { return beamTimeline{} })
	box.DrawOver(func(p *ui.Painter, rect ui.Rect) { paintComposerBeam(p, rect, beam, beamState, !typing, reducedMotion) })
	box.Children(func() {
		paintDraftTextInput(c, Model{Draft: text, TextInput: *client, UseTextInputClient: true, TypingFx: typing, TypingFxStyle: style, OnSubmit: func() {}}, c.Theme().Text, c.Theme().TextMuted, -1, -1).Label("Effect preview")
	})
}
