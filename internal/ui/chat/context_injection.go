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
		if paintToolContext(c, msg, key, open, muted) {
			return
		}
		title := i18n.T(i18n.KeyChatContextTitle)
		if v.Role == "recall" {
			title = i18n.T(i18n.KeyChatContextRecall)
		}
		if v.Producer != "" && v.Producer != "unknown" {
			title += " · " + clipReadOnlyText(v.Producer, 512)
		}
		marker := "▸"
		if *open {
			marker = "▾"
		}
		ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
			if ui.Button(c, marker+" "+title).Label("Context details "+msg.ID).Shrink(1).MinWidth(0).
				FontSize(12).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Clicked() {
				*open = !*open
			}
		})
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
