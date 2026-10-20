package assistant

import (
	"bytes"
	"encoding/json"
	"github.com/egoist/mygo/ui"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"testing"
	"whaleisle/internal/desktop/avatarimage"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/i18n"
)

func TestBotAvatarImageUIExplicitPreviewApplyAndReview(t *testing.T) {
	im := image.NewRGBA(image.Rect(0, 0, 40, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 40; x++ {
			im.SetRGBA(x, y, color.RGBA{180, 40, 100, 255})
		}
	}
	var encoded bytes.Buffer
	png.Encode(&encoded, im)
	source, _ := avatarimage.Decode(encoded.Bytes(), "synthetic.png", "png")
	crop := avatarimage.DefaultCrop("circle")
	preview, _ := source.Bake(crop)
	old := pluginrpc.DshbotAvatarValues{Kind: "blob", Shape: "circle", Color: "blue"}
	raw, _ := json.Marshal(old)
	status := pluginrpc.DshbotCapabilityStatus{Bot: pluginrpc.DshbotCatalogItem{ID: "b", Kind: "bot", SessionID: "s"}, Revision: 8, Profile: map[string]json.RawMessage{"avatar": raw, "workspaceId": json.RawMessage(`"keep-workspace"`), "model": json.RawMessage(`{"provider":"keep-provider","model":"keep-model","reasoningEffort":"high"}`)}}
	state := dshbotstore.AvatarSnapshot{Scope: dshbotstore.RoutineScope{HostKey: "fixture", BotID: "b"}, Loaded: true, Writable: true, Status: status, Draft: old}
	chose, applied, cancelled, saves, changes := 0, 0, 0, 0, 0
	tt := ui.NewTester(func(c *ui.Context) {
		m := BotAvatarModel{State: state, HostLabel: "https://fixture.invalid", OnRefresh: func() {}, OnChooseImage: func() {
			chose++
			state.ImageImport = dshbotstore.AvatarImageImportSnapshot{Ready: true, Name: "synthetic.png", Format: "png", Bytes: encoded.Len(), Width: 40, Height: 20, Crop: crop, Preview: preview}
		}, OnImageCropChange: func(value avatarimage.Crop) {
			changes++
			state.ImageImport.Crop = value
			state.ImageImport.Preview, _ = source.Bake(value)
		}, OnApplyImageCrop: func() {
			applied++
			state.Draft = pluginrpc.DshbotAvatarValues{Kind: "image", DataURL: state.ImageImport.Preview, Crop: state.ImageImport.Crop.Shape}
			state.ImageImport = dshbotstore.AvatarImageImportSnapshot{}
			state.Dirty = true
		}, OnCancelImageCrop: func() { cancelled++; state.ImageImport = dshbotstore.AvatarImageImportSnapshot{} }, OnReview: func() {
			state.Review = &dshbotstore.AvatarConfirmation{Command: pluginrpc.DshbotAvatarCommand{Expected: status, Values: state.Draft}, Sequence: 1}
		}, OnAcknowledge: func(v bool) { state.Review.Acknowledged = v }, OnConfirm: func() { saves++ }}
		botAvatarBlock(c, m, false, ui.RGB(15, 17, 21), ui.RGB(129, 133, 140), ui.RGB(65, 118, 230))
	}, 1100, 4200)
	click := func(label string) {
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	if chose+applied+saves != 0 {
		t.Fatal("render action")
	}
	click(i18n.T(i18n.KeyMiscAssistantChooseLocalAvatarImage))
	if chose != 1 || state.Draft != old || saves != 0 {
		t.Fatal("chooser changed profile")
	}
	for _, label := range []string{"Avatar crop zoom", "Avatar crop X", "Avatar crop Y", i18n.T(i18n.KeyMiscAssistantLocalCropPreview)} {
		if _, ok := tt.Find(label); !ok {
			t.Fatal("missing crop consumer", label)
		}
	}
	r, _ := tt.Find("Avatar crop zoom")
	tt.ClickAt(r.X+r.W*.75, r.Y+r.H/2)
	if changes == 0 || state.ImageImport.Crop.Zoom <= 1 {
		t.Fatal("zoom slider not consumed")
	}
	click(i18n.T(i18n.KeyMiscAssistantCrop) + "square")
	if state.ImageImport.Crop.Shape != "square" {
		t.Fatal("crop display")
	}
	if path := os.Getenv("AVATAR_IMAGE_SCREENSHOT"); path != "" {
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(f, tt.Image())
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	// Transient preview never enters the process-wide saved-image cache.
	botAvatarImageCache.Lock()
	cached := botAvatarImageCache.set
	botAvatarImageCache.Unlock()
	if cached {
		t.Fatal("temporary preview cached")
	}
	click(i18n.T(i18n.KeyMiscAssistantCancelImageSelection))
	if cancelled != 1 || state.Draft != old || state.ImageImport.Ready {
		t.Fatal("cancel")
	}
	click(i18n.T(i18n.KeyMiscAssistantChooseLocalAvatarImage))
	click(i18n.T(i18n.KeyMiscAssistantApplyCropToDraft))
	if applied != 1 || state.Draft.Kind != "image" || saves != 0 {
		t.Fatal("apply saved")
	}
	click(i18n.T(i18n.KeyMiscAssistantReviewAvatar))
	text := strings.Join(tt.Texts(), "\n")
	for _, term := range []string{"192px JPEG", "dataURL", "发送至当前Host：https://fixture.invalid", "不会发送源路径", "原图元数据", "requestSync", "重建已暂停", "工具", "外部消息", "费用", "不自动重试", "SHA-256="} {
		if !strings.Contains(text, term) {
			t.Fatal("missing disclosure", term)
		}
	}
	click(i18n.T(i18n.KeyMiscAssistantConfirmAvatarSave))
	if saves != 0 {
		t.Fatal("missing acknowledgement")
	}
	click(botAvatarAcknowledgement())
	click(i18n.T(i18n.KeyMiscAssistantConfirmAvatarSave))
	if saves != 1 {
		t.Fatal("confirmed save")
	}
}
