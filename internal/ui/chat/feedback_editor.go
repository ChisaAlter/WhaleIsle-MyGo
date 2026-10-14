package chat

import (
	"context"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

func authoritativeFeedbackRow(c *ui.Context, store *session.Store, id string) {
	v := store.Feedback()
	if !v.Ready {
		if v.Loading {
			ui.Text(c, "正在读取反馈 / Loading feedback").FontSize(11)
			return
		}
		if v.Error != "" {
			ui.Text(c, v.Error).FontSize(11)
		}
		if ui.Button(c, "读取反馈 / Load feedback").Clicked() {
			store.RefreshFeedback(context.Background())
		}
		return
	}
	busy := v.BusyID != "" || v.Loading
	item := v.Items[id]
	ui.Row(c).Gap(5).Children(func() {
		for _, rating := range []string{session.MessageFeedbackUp, session.MessageFeedbackDown} {
			rating := rating
			if ui.Button(c, FeedbackButtonLabel(rating, item.Rating)).Disabled(busy).Clicked() {
				if item.Rating == rating {
					store.ClearMessageFeedback(context.Background(), id)
				} else {
					store.BeginFeedback(id, rating)
				}
			}
		}
		if item.Version != "" && ui.Button(c, "编辑反馈 / Edit feedback").Disabled(busy).Clicked() {
			store.BeginFeedback(id, item.Rating)
		}
	})
	if v.Error != "" {
		ui.Text(c, v.Error).FontSize(11)
	}
	if v.EditorID != id {
		return
	}
	ui.Text(c, "此条消息 / This message: "+id+" · "+v.Rating).FontSize(11)
	note := v.Note
	ui.TextArea(c, &note).FillWidth().Height(80).Disabled(busy).OnChange(func() { store.EditFeedback(note, v.Category) })
	ui.Row(c).Wrap().Gap(4).Children(func() {
		for _, category := range []string{"", "task-result", "instruction-following", "product-interaction", "service-stability", "resource-cost", "security-privacy-permission", "other"} {
			category := category
			label := category
			if label == "" {
				label = "无分类 / None"
			}
			if category == v.Category {
				label = "● " + label
			}
			if ui.Button(c, label).Disabled(busy).Clicked() {
				store.EditFeedback(note, category)
			}
		}
	})
	ui.Row(c).Gap(6).Children(func() {
		if ui.Button(c, "提交这条反馈 / Submit feedback").Disabled(busy).Clicked() {
			store.SubmitFeedback(context.Background())
		}
		if ui.Button(c, "取消 / Cancel").Disabled(busy).Clicked() {
			store.CancelFeedback()
		}
	})
}
