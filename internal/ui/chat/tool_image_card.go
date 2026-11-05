package chat

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Keyed registrants own their result material; only the generic fallback supplies
// genericImageCardModel. In particular a malformed read_image keeps its OUT body.
func toolUsesGenericImageCard(name string) bool {
	switch name {
	case "read", "read_image", "write", "edit", "grep", "glob", "web_search", "web_fetch", "bash", "run_code", "todo_write", "ask_user_question":
		return false
	}
	return !toolHasDetailsView(name)
}

func toolHasDetailsView(name string) bool {
	switch name {
	case "create_goal", "get_goal", "update_goal", "schedule_create", "schedule_list", "schedule_update", "schedule_delete", "job_list", "job_output", "job_kill", "terminal_open", "terminal_read", "terminal_list", "terminal_signal", "terminal_close", "session_event_read", "session_event_search", "session_event_trace", "session_search", "session_trace", "list_subagent_models", "subagent", "list_agents", "send_message", "interrupt_agent", "spawn_teammate", "team_task_create", "team_task_get", "team_task_update", "team_task_list", "wait_agent", "workflow", "ralph", "cordis_inspect_list", "cordis_inspect_query", "cordis_inspect_self", "lsp":
		return true
	}
	return false
}

func toolImageLoaderAvailable() bool {
	return attachmentImageRead().Fetch != nil
}

func toolImageCard(c *ui.Context, model *toolImageModel, st ToolCardStyle, key string, onOpen func(string), renderers ...func(func())) {
	ui.Column(c.Key("tool-image:"+key)).FillWidth().MinWidth(0).Margin(4, 0, 4, 4).Children(func() {
		gallery := func() {
			ui.Row(c.Key("gallery")).Wrap().Gap(10).MaxWidthPercent(100).AlignSelf(ui.Start).Children(func() {
				for index, image := range model.Images {
					toolImageFrame(c.Key(strconv.Itoa(index)+":"+image.AttachmentID), image, len(model.Images) > 1, st, onOpen)
				}
			})
		}
		if len(renderers) > 0 && renderers[0] != nil {
			renderers[0](gallery)
		} else {
			gallery()
		}
		font := theme.RenderPreferences().Code.Family
		ui.Column(c.Key("metadata")).FillWidth().MinWidth(0).Margin(4, 0, 0, 0).Children(func() {
			if model.HasPath {
				toolImageCardText(c, model.Path, font, 11, 16, st.Muted)
			}
			for _, image := range model.Images {
				toolImageCardText(c, toolImageSizeText(image), font, 11, 16, st.Muted)
			}
		})
		if model.ExtraText != nil {
			ui.Column(c.Key("extra")).FillWidth().MinWidth(0).Margin(4, 0, 0, 0).Children(func() {
				toolImageCardText(c, *model.ExtraText, theme.RenderPreferences().Chat.Family, 13, 20, st.Muted)
			})
		}
	})
}

// Keep provider text, spaces and line endings available to selection verbatim.
func toolImageCardText(c *ui.Context, text, font string, size, height float32, ink ui.Color) {
	if text == "" {
		return
	}
	ui.Text(c, text).Font(font).FontSize(size).FixedLineHeight(height).TextColor(ink).MinWidth(0).Selectable()
}

func toolImageNumber(n float64) string {
	value, _ := contextRawNumber([]byte(strconv.FormatFloat(n, 'g', -1, 64)))
	return value
}

func toolImageSizeText(image toolImageAttachment) string {
	w, h, bytes := toolImageNumber(image.Width), toolImageNumber(image.Height), toolImageNumber(image.Bytes)
	if original := image.OriginalDimensions; original != nil {
		return fmt.Sprintf(i18n.T(i18n.KeyToolImageDownscaled), w, h, toolImageNumber(original.Width), toolImageNumber(original.Height), bytes)
	}
	return fmt.Sprintf(i18n.T(i18n.KeyToolImageSize), w, h, bytes)
}

// The durable dimensions reserve the thumbnail before decoding; CSS minimum44
// applies after singleFit's round-to-nearest size, including tiny natural images.
func toolImageFit(width, height float64, tile bool) (float32, float32) {
	if tile {
		return 64, 64
	}
	ratio := math.Min(4, math.Max(.25, width/height))
	w, h := float64(240), 240/ratio
	if ratio < 1 {
		w, h = 240*ratio, 240
	}
	scale := math.Min(1, math.Min(width/w, height/h))
	return float32(math.Max(44, math.Round(w*scale))), float32(math.Max(44, math.Round(h*scale)))
}

// Cached and loaded bytes use the same session-authorized attachment loader as
// chat images. The recorded source path is never read as an image or retried.
func toolImageFrame(c *ui.Context, image toolImageAttachment, tile bool, st ToolCardStyle, onOpen func(string)) {
	read := attachmentImageRead()
	fetch := read.Fetch
	path := read.Cached(image.AttachmentID)
	var bitmap *ui.Bitmap
	if path != "" {
		bitmap = loadChatBitmap(path)
	}
	state := attachmentLoad{}
	if bitmap == nil && fetch != nil {
		svc := c.Services()
		state = attachmentLoads.ensure(read.Identity(image.AttachmentID), func(ctx context.Context, _ string) (string, error) {
			local, err := fetch(ctx, image.AttachmentID)
			if err == nil && loadFetchedChatBitmap(local) == nil {
				return "", fmt.Errorf("invalid or unreadable image")
			}
			return local, err
		}, svc.Invalidate)
		if state.path != "" {
			path, bitmap = state.path, loadChatBitmap(state.path)
		}
	}
	border := float32(math.Max(1, math.Floor(.5*float64(c.DeviceScale()))) / float64(c.DeviceScale()))
	hover, borderInk := st.Field, st.Border
	if st.Colors.Hover != "" {
		hover, borderInk = chrome.Color(st.Colors.Hover), chrome.Color(st.Colors.Border)
		if darkSurface(st.FG) {
			borderInk = chrome.Color(st.Colors.BorderL1)
		}
	}
	if state.err != nil {
		toolImageRetry(c, image.AttachmentID, tile, border, borderInk, st)
		return
	}
	w, h := toolImageFit(image.Width, image.Height, tile)
	label := i18n.T(i18n.KeyToolImageLabel)
	if image.Name != nil {
		label = *image.Name
	}
	frame := ui.ButtonBase(c).Width(w).Height(h).MinWidth(w).MinHeight(h).Shrink(0).Padding(0).
		Radius(20).Border(border, borderInk).Background(hover).Center().FocusRing(false).
		Label(fmt.Sprintf(i18n.T(i18n.KeyToolImageOpenNamed), label)).Tooltip(i18n.T(i18n.KeyToolImageOpen)).Cursor(ui.CursorPointer)
	if frame.Clicked() && bitmap != nil && onOpen != nil {
		onOpen(path)
	}
	focused := frame.FocusVisible()
	frame.DrawOver(func(p *ui.Painter, r ui.Rect) {
		if bitmap != nil {
			inner := ui.Rect{X: r.X + border, Y: r.Y + border, W: max(0, r.W-2*border), H: max(0, r.H-2*border)}
			destination := toolImageCrop(bitmap, inner, image.Width/image.Height, tile)
			p.Clip(inner, 20-border, func() { p.Image(bitmap, destination, ui.FillBox) })
		}
		if focused {
			p.Stroke(ui.Rect{X: r.X - 4, Y: r.Y - 4, W: r.W + 8, H: r.H + 8}, st.Accent, 24, 2)
		}
	})
	if bitmap == nil {
		frame.Children(func() {
			ui.Text(c, i18n.T(i18n.KeyToolImageLoading)).Font(theme.RenderPreferences().Chat.Family).FontSize(12).FixedLineHeight(18).TextColor(st.Muted).PassThrough()
		})
	}
}

func toolImageCrop(bitmap *ui.Bitmap, box ui.Rect, natural float64, tile bool) ui.Rect {
	bw, bh := bitmap.Size()
	scale := max(box.W/float32(bw), box.H/float32(bh))
	w, h := float32(bw)*scale, float32(bh)*scale
	x, y := box.X+(box.W-w)/2, box.Y+(box.H-h)/2
	if !tile && natural < .25 {
		y = box.Y
	}
	if !tile && natural > 4 {
		x = box.X
	}
	return ui.Rect{X: x, Y: y, W: w, H: h}
}

func toolImageRetry(c *ui.Context, id string, tile bool, border float32, borderInk ui.Color, st ToolCardStyle) {
	read := attachmentImageRead()
	// --dsw-alias-interactive-bg-hover-danger: mix of the error ink and transparent.
	failure := ui.Hex("#ec13130d")
	if darkSurface(st.FG) {
		failure = ui.Hex("#f25a5a26")
	}
	button := ui.ButtonBase(c).MaxWidth(240).MinWidth(0).Padding(10, 12, 10, 12).Radius(12).
		Border(border, borderInk).Background(failure).TextColor(st.Muted).Label(i18n.T(i18n.KeyToolImageFailed)).Cursor(ui.CursorPointer).FocusRing(false)
	if tile {
		button.Size(64, 64).MinWidth(64).MinHeight(64).Padding(4).Radius(20).Clip()
	}
	if button.Clicked() {
		read.Forget(id)
		attachmentLoads.retry(read.Identity(id))
		c.Invalidate()
	}
	focused := button.FocusVisible()
	button.DrawOver(func(p *ui.Painter, r ui.Rect) {
		if focused {
			p.Stroke(ui.Rect{X: r.X - 4, Y: r.Y - 4, W: r.W + 8, H: r.H + 8}, st.Accent, 12, 2)
		}
	})
	button.Children(func() {
		ui.Text(c, i18n.T(i18n.KeyToolImageFailed)).Font(theme.RenderPreferences().Chat.Family).FontSize(12).FixedLineHeight(18).TextColor(st.Muted).PassThrough()
	})
}
