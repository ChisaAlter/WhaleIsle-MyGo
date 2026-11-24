package openui

import (
	"errors"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/errcopy"
	"whaleisle/internal/openui/lang"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

// webOnly are the components that stay on the web engine in the daily path
// (ARCHITECTURE.md §7): they open only in the A2 compatibility workbench,
// after a click.
var webOnly = map[string]bool{"HtmlView": true, "YouTube": true}

// OpenCompatLabel is the placeholder action.
func OpenCompatLabel() string {
	return i18n.Pick("在兼容工作台中打开", "Open in compatibility workbench")
}

// placeholder is the compact stand-in of a component without a native view:
// what it is and the explicit action that opens the card in the A2
// workbench. It never starts a browser by itself.
func placeholder(r *Render, el *lang.Element) ui.Element {
	t := r.T
	title := str(el, "title")
	desc := i18n.Pick("此组件需要在兼容工作台中显示。", "This component shows in the compatibility workbench.")
	icon := "layout-template"
	switch el.Type {
	case "YouTube":
		icon = "youtube"
		if title == "" {
			title = i18n.Pick("视频", "Video")
		}
		desc = i18n.Pick("视频在兼容工作台中播放。", "The video plays in the compatibility workbench.")
	case "HtmlView":
		icon = "code-xml"
		if title == "" {
			title = i18n.Pick("网页内容", "Web content")
		}
		desc = i18n.Pick("网页内容在兼容工作台中运行。", "The web content runs in the compatibility workbench.")
	}
	if title == "" {
		title = el.Type
	}
	return ui.Row(r.C).FillWidth().MinWidth(0).Gap(design.Sp(3)).AlignItems(ui.Center).Padding(design.Sp(3), design.Sp(3.5)).
		Radius(design.RadiusXL).Border(design.BorderWidth, t.Border).Background(t.Muted.Alpha(0.5)).Children(func() {
		r.icon(icon, design.IconLG, t.MutedForeground)
		ui.Column(r.C).Grow(1).MinWidth(0).Gap(design.Sp(0.5)).Children(func() {
			r.paragraph(title, design.TextSM, design.WeightMedium, t.Foreground)
			r.paragraph(desc, design.TextXS, design.WeightNormal, t.MutedForeground)
		})
		if r.opts.OpenCompat != nil {
			if coss.Button(r.C, OpenCompatLabel(), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Disabled: r.Streaming()}).Clicked() {
				r.opts.OpenCompat()
			}
		}
	})
}

// loading is the card while nothing renders yet.
func loading(r *Render) {
	ui.Column(r.C).FillWidth().Gap(design.Sp(2)).Children(func() {
		coss.Skeleton(r.C).Height(design.Sp(5)).Width(design.Sp(40))
		coss.Skeleton(r.C).Height(design.Sp(16)).FillWidth()
	})
}

// diagnosticError classifies a frame's diagnostics into the errcopy family
// whose sentence the card shows; the raw diagnostics are never painted.
func diagnosticError(errs []lang.Diagnostic) error {
	for _, d := range errs {
		if d.Code == lang.CodeBudgetExceeded || d.Code == lang.CodeProgramTooLarge || d.Code == lang.CodeStateTooLarge {
			return errors.New("whale ui program over budget")
		}
	}
	return errors.New("whale ui program invalid")
}

// frameErrors reports a settled frame that has problems. A frame without a
// root shows the failure sentence; one that renders shows nothing more,
// since its dropped parts already fell back (spec §8 recovery).
func frameErrors(r *Render, f *lang.Frame) {
	if f.Root != nil || len(f.Errors) == 0 {
		return
	}
	actions := func() {}
	if r.opts.OpenCompat != nil {
		actions = func() {
			if coss.Button(r.C, OpenCompatLabel(), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM}).Clicked() {
				r.opts.OpenCompat()
			}
		}
	}
	coss.Alert(r.C, "", errcopy.Sentence(diagnosticError(f.Errors)), coss.AlertOptions{Variant: coss.AlertWarning, Action: actions})
}

// notice shows a card-level failure (state, panel, media) as a sentence.
func notice(r *Render, err error) {
	coss.Alert(r.C, "", errcopy.Sentence(err), coss.AlertOptions{Variant: coss.AlertError})
}

// panelNote stands in for a spec that opened as the conversation panel.
func panelNote(r *Render) {
	ui.Row(r.C).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
		r.icon("panel-right", design.IconSM, r.T.MutedForeground)
		r.paragraph(i18n.Pick("已显示在会话面板中。", "Shown in the conversation panel."), design.TextSM, design.WeightNormal, r.T.MutedForeground)
	})
}
