package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

func botAvatarImageImportBlock(c *ui.Context, m BotAvatarModel, eligible bool, fg, muted, accent ui.Color) {
	p := m.State.ImageImport
	if ui.Button(c, i18n.T(i18n.KeyMiscAssistantChooseLocalAvatarImage)).Disabled(!eligible || p.Busy || p.Ready || m.OnChooseImage == nil).Clicked() {
		m.OnChooseImage()
	}
	ui.Text(c, "仅手动选择本地 JPEG/PNG，≤8 MiB、边长≤4096、≤800万像素。裁剪输出192×192 JPEG（质量82），不保留原图元数据。此处仅本地预览，确认保存才随完整资料发送当前Host。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	if p.Error != "" {
		ui.Text(c, p.Error).TextColor(accent)
	}
	if !p.Busy && !p.Ready {
		return
	}
	ui.Column(c.Key("avatar-image-crop")).Border(1, accent).Radius(8).Padding(10).Gap(6).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantLocalImageCrop)).Bold().TextColor(fg)
		if p.Busy {
			ui.Text(c, "等待选择或在后台读取解码；取消、隐藏或切换Host/Bot会丢弃这次选择。").TextColor(muted)
		}
		if p.Ready {
			ui.Text(c, fmt.Sprintf("%s · %s · %d bytes · %d×%d", p.Name, p.Format, p.Bytes, p.Width, p.Height)).TextColor(fg).Selectable()
			value := pluginrpc.DshbotAvatarValues{Kind: "image", DataURL: p.Preview, Crop: p.Crop.Shape}
			_ = botAvatarUncachedPreview(c.Key("crop-preview"), value, 192, i18n.T(i18n.KeyMiscAssistantLocalCropPreview), fg, muted)
			ui.Text(c, "使用缩放与X/Y位置滑条（本轮替代原版拖动）；圆形/方形仅改变显示裁剪，输出像素始终为方形。").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			crop := p.Crop
			changed := false
			ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantZoomPercent), crop.Zoom)).TextColor(fg)
			changed = ui.StepSlider(c.Key("crop-zoom"), &crop.Zoom, 1, 3, .01).Label("Avatar crop zoom").Disabled(!eligible).Changed() || changed
			ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantHorizontalOffset), crop.X)).TextColor(fg)
			changed = ui.StepSlider(c.Key("crop-x"), &crop.X, 0, 1, .01).Label("Avatar crop X").Disabled(!eligible).Changed() || changed
			ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantVerticalOffset), crop.Y)).TextColor(fg)
			changed = ui.StepSlider(c.Key("crop-y"), &crop.Y, 0, 1, .01).Label("Avatar crop Y").Disabled(!eligible).Changed() || changed
			for _, shape := range []string{"circle", "square"} {
				b := ui.Button(c, i18n.T(i18n.KeyMiscAssistantCrop)+shape).Disabled(!eligible)
				if crop.Shape == shape {
					b.Border(2, accent)
				}
				if b.Clicked() {
					crop.Shape = shape
					changed = true
				}
			}
			if changed && m.OnImageCropChange != nil {
				m.OnImageCropChange(crop)
			}
			if ui.Button(c, i18n.T(i18n.KeyMiscAssistantApplyCropToDraft)).Disabled(!eligible || p.Error != "" || m.OnApplyImageCrop == nil).Clicked() {
				m.OnApplyImageCrop()
			}
		}
		if ui.Button(c, i18n.T(i18n.KeyMiscAssistantCancelImageSelection)).Disabled(m.OnCancelImageCrop == nil).Clicked() {
			m.OnCancelImageCrop()
		}
	})
}

// Draft/crop/review-after images have no process-wide thumbnail cache. Cancel
// or hide releases their model/source references; only saved images use cache.
func botAvatarUncachedPreview(c *ui.Context, v pluginrpc.DshbotAvatarValues, size float32, label string, fg, muted ui.Color) error {
	if v.Kind != "image" {
		return botAvatarPreview(c, v, size, label, fg, muted)
	}
	bmp, err := decodeBotAvatarImage(v)
	if err != nil {
		return err
	}
	radius := size * 10 / 64
	if v.Crop == "circle" {
		radius = size / 2
	}
	ui.Image(c, bmp).Size(size, size).Fit(ui.Cover).Radius(radius).Label(label)
	return nil
}
