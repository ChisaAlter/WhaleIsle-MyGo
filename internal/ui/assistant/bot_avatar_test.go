package assistant

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"github.com/egoist/mygo/ui"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/i18n"
)

func avatarImageFixture(w, h int) pluginrpc.DshbotAvatarValues {
	im := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			im.SetRGBA(x, y, color.RGBA{180, 60, 140, 255})
		}
	}
	var b bytes.Buffer
	png.Encode(&b, im)
	return pluginrpc.DshbotAvatarValues{Kind: "image", DataURL: "data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes()), Crop: "circle"}
}
func TestBotAvatarBoundedInlineDecodeAndPixels(t *testing.T) {
	v := avatarImageFixture(16, 16)
	if _, err := decodeBotAvatarImage(v); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []pluginrpc.DshbotAvatarValues{avatarImageFixture(2049, 1), avatarImageFixture(1025, 1025), {Kind: "image", Crop: "circle", DataURL: "data:image/png;base64,!bad"}, {Kind: "image", Crop: "circle", DataURL: strings.Replace(v.DataURL, "image/png", "image/jpeg", 1)}, {Kind: "image", Crop: "circle", DataURL: "https://fixture.invalid/avatar.png"}, {Kind: "image", Crop: "circle", DataURL: "data:image/png;base64," + strings.Repeat("A", 120001)}} {
		if _, err := decodeBotAvatarImage(bad); err == nil {
			t.Fatal("unsafe image accepted")
		}
	}
	fg, muted := ui.RGB(15, 17, 21), ui.RGB(129, 133, 140)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Gap(5).Children(func() {
			ui.Row(c).Children(func() {
				for _, shape := range pluginrpc.DshbotAvatarShapes() {
					botAvatarDrawBlob(c, pluginrpc.DshbotAvatarValues{Kind: "blob", Shape: shape, Color: "blue"}, 64, "shape-"+shape, fg, muted)
				}
			})
			ui.Row(c).Children(func() {
				for _, color := range pluginrpc.DshbotAvatarColors() {
					botAvatarDrawBlob(c, pluginrpc.DshbotAvatarValues{Kind: "blob", Shape: "circle", Color: color}, 64, "color-"+color, fg, muted)
				}
			})
			_ = botAvatarPreview(c, v, 64, "image-circle", fg, muted)
		})
	}, 800, 240)
	pixels := tt.Image()
	hashes := map[[32]byte]bool{}
	for _, shape := range pluginrpc.DshbotAvatarShapes() {
		r, ok := tt.Find("shape-" + shape)
		if !ok {
			t.Fatal(shape)
		}
		var data bytes.Buffer
		for y := int(r.Y); y < int(r.Y+r.H); y++ {
			for x := int(r.X); x < int(r.X+r.W); x++ {
				p := pixels.RGBAAt(x, y)
				data.Write([]byte{p.R, p.G, p.B, p.A})
			}
		}
		sum := sha256.Sum256(data.Bytes())
		if hashes[sum] {
			t.Fatal("shape not rendered distinctly", shape)
		}
		hashes[sum] = true
	}
	for _, name := range pluginrpc.DshbotAvatarColors() {
		r, _ := tt.Find("color-" + name)
		got := pixels.RGBAAt(int(r.X+32), int(r.Y+45))
		want := botAvatarColor(name, fg, muted)
		if got.R != want.R || got.G != want.G || got.B != want.B {
			t.Fatal("color not consumed", name, got, want)
		}
	}
	r, _ := tt.Find("shape-circle")
	eye := pixels.RGBAAt(int(r.X+23), int(r.Y+29))
	if eye.R < 240 || eye.G < 240 || eye.B < 240 {
		t.Fatal("eyes missing", eye)
	}
	r, _ = tt.Find("image-circle")
	center := pixels.RGBAAt(int(r.X+32), int(r.Y+32))
	corner := pixels.RGBAAt(int(r.X), int(r.Y))
	if center.R != 180 || center.B != 140 || corner == center {
		t.Fatal("inline image or circle crop not consumed", center, corner)
	}
	if path := os.Getenv("AVATAR_FIXTURE_SCREENSHOT"); path != "" {
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(f, pixels)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
}
func TestBotAvatarUIExplicitImageReplacementAndImpact(t *testing.T) {
	old := avatarImageFixture(8, 8)
	raw, _ := json.Marshal(old)
	status := pluginrpc.DshbotCapabilityStatus{Bot: pluginrpc.DshbotCatalogItem{ID: "b", Kind: "bot", SessionID: "s"}, Revision: 8, Profile: map[string]json.RawMessage{"avatar": raw, "workspaceId": json.RawMessage(`"keep-workspace"`), "model": json.RawMessage(`{"provider":"keep-provider","model":"keep-model","reasoningEffort":"high"}`)}}
	state := dshbotstore.AvatarSnapshot{Scope: dshbotstore.RoutineScope{HostKey: "fixture-account", BotID: "b"}, Loaded: true, Writable: true, Status: status, Draft: old}
	reads, saves := 0, 0
	tt := ui.NewTester(func(c *ui.Context) {
		m := BotAvatarModel{State: state, HostLabel: "https://fixture.invalid", OnRefresh: func() { reads++ }, OnChange: func(v pluginrpc.DshbotAvatarValues) { state.Draft = v; state.Dirty = true }, OnReview: func() {
			state.Review = &dshbotstore.AvatarConfirmation{Command: pluginrpc.DshbotAvatarCommand{Expected: status, Values: state.Draft}, Sequence: 1}
		}, OnAcknowledge: func(a bool) { state.Review.Acknowledged = a }, OnConfirm: func() { saves++ }}
		botAvatarBlock(c, m, false, ui.RGB(15, 17, 21), ui.RGB(129, 133, 140), ui.RGB(65, 118, 230))
	}, 1100, 3600)
	if reads != 0 || saves != 0 || state.Draft != old || state.Dirty {
		t.Fatal("render coerced image or dispatched")
	}
	if err := tt.Click(i18n.T(i18n.KeyMiscAssistantShape) + "cloud"); err != nil {
		t.Fatal(err)
	}
	if state.Draft.Kind != "blob" || state.Draft.Shape != "cloud" || state.Draft.Color != "ink" {
		t.Fatal("explicit shape choice", state.Draft)
	}
	if err := tt.Click(i18n.T(i18n.KeyMiscAssistantColor) + "teal"); err != nil {
		t.Fatal(err)
	}
	if state.Draft.Shape != "cloud" || state.Draft.Color != "teal" {
		t.Fatal("color did not preserve shape")
	}
	if err := tt.Click(i18n.T(i18n.KeyMiscAssistantReviewAvatar)); err != nil {
		t.Fatal(err)
	}
	text := strings.Join(tt.Texts(), "\n")
	for _, term := range []string{"kind=image", "crop=circle", "SHA-256=", "kind=blob", "shape=cloud", "color=teal", "revision 8", "替换当前内嵌图片", "presentation", "rename", "requestSync", "重建已暂停", "模型", "工具", "外部消息", "费用", "不主动 prompt/start/resume/open", "不自动重试", "Bot Web", "keep-workspace", "keep-provider"} {
		if !strings.Contains(text, term) {
			t.Fatal("missing review", term)
		}
	}
	tt.Click(i18n.T(i18n.KeyMiscAssistantConfirmAvatarSave))
	if saves != 0 {
		t.Fatal("unacknowledged")
	}
	if err := tt.Click(botAvatarAcknowledgement()); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(i18n.T(i18n.KeyMiscAssistantConfirmAvatarSave)); err != nil {
		t.Fatal(err)
	}
	if saves != 1 {
		t.Fatal("explicit save")
	}
}
