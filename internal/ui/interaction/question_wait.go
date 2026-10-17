package interaction

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/interaction"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func questionWaitStatus(c *ui.Context, p interaction.Pending, card interaction.QuestionCardView, store *interaction.Store, muted ui.Color) {
	w := card.Wait
	if w == nil || p.Continued {
		return
	}
	zh := i18n.Locale() == i18n.LocaleZH
	message := ""
	held := w.State == "waiting" || w.State == "editing"
	if held {
		if zh {
			message = "已暂停倒计时，等待你回答。"
		} else {
			message = "Countdown held while you answer."
		}
	} else {
		seconds := (w.RemainingMs + 999) / 1000
		if w.Running {
			if zh {
				message = fmt.Sprintf("%d 秒后继续", seconds)
			} else {
				message = fmt.Sprintf("Continuing in %d seconds", seconds)
			}
		} else {
			if zh {
				message = fmt.Sprintf("倒计时已暂停：剩余 %d 秒", seconds)
			} else {
				message = fmt.Sprintf("Countdown paused: %d seconds remaining", seconds)
			}
		}
	}
	ui.Row(c).FillWidth().Gap(8).Children(func() {
		ui.Text(c, message).FontSize(theme.InterfaceSize(11)).TextColor(muted).Grow(1)
		if !held {
			label := "Take my time"
			if zh {
				label = "让我慢慢回答"
			}
			if ui.Button(c, label).Disabled(p.Sending).Clicked() {
				store.TakeQuestionTime(p.AgentID, p.EventID)
			}
		}
	})
}
