package frame

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"
)

var inspectorCopyIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2"><rect x="5.5" y="5.5" width="8" height="8" rx="1.5"/><path d="M10.5 3V2.5h-8v8H3"/></svg>`))

func inspectorDetails(c *ui.Context, p *SessionInspectorPaneState, object inspectorObject, objects map[string]inspectorObject, rows []session.InspectorRecord, available float32, colors theme.Colors) {
	t := tk(c)
	ink, border := t.Foreground, t.Border
	ui.Column(c.Key("inspector-details-panel")).Height(available * p.inspectorDetailsPercent() / 100).Shrink(0).MinHeight(0).FillWidth().FontSize(11).TextColor(ink).Children(func() {
		inspectorDetailsDivider(c, p, available)
		ui.Row(c.Key("inspector-details-toolbar")).FillWidth().Gap(design.Sp(2)).Padding(design.Sp(1.5), design.Sp(2)).AlignItems(ui.Center).BorderWidth(0, 0, design.BorderWidth/2, 0).BorderColor(border).Children(func() {
			title := paneLabel("原始数据", "Raw data")
			if p.Mode == "chat-node" {
				title = object.Label + " · " + object.Identity
			}
			inspectorText(c, title).Grow(1)
			// Retain the native copy command as a small secondary action.
			copyHint := paneLabel("复制原始 JSON", "Copy raw JSON")
			if object.JSONError != "" {
				copyHint = paneLabel("无法导出 JSON：", "JSON export unavailable: ") + object.JSONError
			}
			copy := inspectorButton(c, paneLabel("复制原始 JSON", "Copy raw JSON")).Tooltip(copyHint).Size(16, 16).Disabled(object.Missing || object.JSONError != "")
			copy.Children(func() { ui.Icon(c, inspectorCopyIcon).Size(13, 13).TextColor(ink) })
			if copy.Clicked() {
				mygo.Clipboard.WriteText(string(object.Raw))
			}
			label := paneLabel("关闭", "Close")
			close := inspectorButton(c, label)
			close.Children(func() { inspectorText(c, label) })
			if close.Clicked() {
				p.closeInspectorDetails()
			}
		})
		coss.ScrollArea(c.Key("inspector-details:"+p.binding+":"+object.Key), coss.ScrollAreaOptions{}, func() {
			if object.Missing {
				ui.Text(c, paneLabel("该记录已移除或结算，请选择当前记录。", "This record was removed or settled. Select a current record.")).FontSize(11)
				return
			}
			if p.Mode == "chat-node" {
				ui.Row(c).Gap(design.Sp(1.5)).FillWidth().Padding(design.Sp(1.5), 0).Margin(0, 0, design.Sp(2), 0).BorderWidth(0, 0, design.BorderWidth/2, 0).BorderColor(border).Children(func() {
					for i, path := range p.breadcrumb {
						if i > 0 {
							inspectorText(c, "›")
						}
						label := path
						if next, ok := objects[path]; ok {
							label = next.Label + " · " + next.Identity
						}
						current := i == len(p.breadcrumb)-1
						button := inspectorButton(c, label).Tooltip(label).MaxWidth(180).Disabled(current)
						color := t.Primary
						if current {
							color = ink
						}
						button.Children(func() { inspectorText(c, label).TextColor(color) })
						if button.Clicked() {
							p.backInspectorReference(i, rows)
						}
					}
				})
				inspectorTypedTree(c, &p.objectTree, object, objects, func(key string) { p.followInspectorReference(key, rows) }, colors)
				return
			}
			inspectorJSONTree(c, &p.tree, object.Raw, colors)
		}).Grow(1).MinHeight(0).FillWidth().Padding(design.Sp(2))
	})
}
