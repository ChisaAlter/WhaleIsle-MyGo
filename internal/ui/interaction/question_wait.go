package interaction

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/interaction"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

func questionWaitStatus(c *ui.Context, p interaction.Pending, card interaction.QuestionCardView, store CardStore, colors theme.Colors, muted ui.Color) {
	zh := i18n.Locale() == i18n.LocaleZH
	if p.Continued {
		message := "Work continued — you can still answer"
		if zh {
			message = "已继续工作，仍可回答"
		}
		ui.Row(c).FillWidth().Gap(8).Children(func() {
			ui.Text(c, message).FontSize(theme.InterfaceSize(11)).TextColor(muted).Grow(1)
		})
		return
	}
	w := card.Wait
	if w == nil {
		return
	}
	message := ""
	held := w.State == "waiting" || w.State == "editing"
	if held {
		if zh {
			message = "会一直等你回答"
		} else {
			message = "Will wait for your answer."
		}
	} else {
		seconds := (w.RemainingMs + 999) / 1000
		if w.Running {
			if zh {
				message = fmt.Sprintf("%d 秒后继续工作", seconds)
			} else {
				message = fmt.Sprintf("Continuing in %d seconds", seconds)
			}
		} else {
			if zh {
				message = fmt.Sprintf("已暂停 · 剩余 %d 秒", seconds)
			} else {
				message = fmt.Sprintf("Paused · %d seconds remaining", seconds)
			}
		}
	}
	ui.Row(c).FillWidth().Gap(8).Children(func() {
		ui.Text(c, message).FontSize(theme.InterfaceSize(11)).TextColor(muted).Grow(1)
		if !held {
			if widgets.Button(c, colors, i18n.T(i18n.KeyChatQuestionTakeTime), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall, Disabled: p.Sending}).Clicked() {
				store.TakeQuestionTime(p.AgentID, p.EventID)
			}
		}
	})
}
