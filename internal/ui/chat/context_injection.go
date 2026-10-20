package chat

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// ContextInjectionView is deliberately read-only. Producer metadata never
// becomes an instruction, actionable path or an ordinary human-message bubble.
func ContextInjectionView(c *ui.Context, msg session.Message, scope string, fg, muted ui.Color) {
	v := session.ContextMessageDetails(msg)
	key := fmt.Sprintf("context:%s:%s:%g", scope, msg.ID, msg.Seq)
	box := ui.Column(c.Key(key)).FillWidth().Gap(4)
	open := ui.Local(box, "open", func() bool { return false })
	box.Children(func() {
		title := historyNavigationText(i18n.T(i18n.KeyChatContextTitle), "Injected context")
		if v.Role == "recall" {
			title = historyNavigationText(i18n.T(i18n.KeyChatContextRecall), "Recalled context")
		}
		title += " · " + clipReadOnlyText(v.Producer, 512)
		if v.Form != "" {
			title += " · " + v.Form
		}
		if ui.Button(c, title).Label("Context details " + msg.ID).FillWidth().Clicked() {
			*open = !*open
		}
		if !*open {
			return
		}
		ui.Scroll(c).FillWidth().MaxHeight(240).Children(func() {
			ui.Column(c).FillWidth().Gap(4).Children(func() {
				// A bounded viewport/text budget affects display only. Copy retains all
				// model-facing text and metadata, and the journal is never rewritten.
				budget := 64 << 10
				paint := func(label, body string) {
					if budget <= 0 {
						return
					}
					shown := clipReadOnlyText(body, budget)
					if len(body) > budget {
						budget = 0
					} else {
						budget -= len(body)
					}
					ui.Text(c, label).FontSize(11).TextColor(muted)
					ui.Text(c, shown).FontSize(12).TextColor(fg).Selectable()
				}
				paint(historyNavigationText(i18n.T(i18n.KeyChatContextContent), "Model-facing content"), v.Body)
				for _, field := range v.Fields {
					paint(field.Name, field.Value)
				}
				if budget <= 0 {
					ui.Text(c, historyNavigationText(i18n.T(i18n.KeyChatContextClipped), "Display clipped; copy includes full content.")).FontSize(11).TextColor(muted)
				}
			})
		})
		if ui.Button(c, historyNavigationText(i18n.T(i18n.KeyChatContextCopy), "Copy context")).Clicked() {
			var parts []string
			parts = append(parts, v.Body)
			for _, field := range v.Fields {
				parts = append(parts, field.Name+": "+field.Value)
			}
			c.Services().WriteClipboard(strings.Join(parts, "\n"))
		}
	})
}
