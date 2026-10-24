package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func toolImageViewCall(t *testing.T, generic bool, count int, extra *string) *session.ToolCallView {
	t.Helper()
	content := []any{}
	name := "read_image"
	if generic {
		name = "render_plot"
	} else {
		content = append(content, map[string]any{"type": "text", "text": toolImageModelEnvelopeForTest})
	}
	for i := 0; i < count; i++ {
		content = append(content, map[string]any{"type": "image", "attachment": map[string]any{"attachmentId": "fixture-id-" + string(rune('a'+i)), "mediaType": "image/png", "bytes": 2048, "width": 480, "height": 240, "name": "sample-" + string(rune('a'+i)), "originalDimensions": map[string]any{"width": 960, "height": 480}}})
	}
	if extra != nil {
		content = append(content, map[string]any{"type": "text", "text": *extra})
	}
	return toolImageModelCallForTest(t, false, name, `{"file_path":"/work/argument.png"}`, `{"path":"/work/recorded.png"}`, toolImageModelJSONForTest(t, content))
}

func toolImageViewFixture(t *testing.T, call *session.ToolCallView, dark, textOnly bool, width int) (*ui.Tester, *map[string]bool, *[]string, *[]string) {
	t.Helper()
	oldLocale, oldPrefs := i18n.Locale(), theme.CurrentPreferences()
	i18n.SetLocale(i18n.LocaleEN)
	theme.SetPreferences(theme.Defaults())
	t.Cleanup(func() { i18n.SetLocale(oldLocale); theme.SetPreferences(oldPrefs) })
	expanded := map[string]bool{}
	opened, files := []string{}, []string{}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.ApplyInterface(c)
		palette := theme.Resolve(theme.DefaultID).Active(dark)
		ui.Column(c).Fill().Padding(20).Background(chrome.Color(palette.BG)).Children(func() {
			ToolMessageView(c, session.Message{Tool: call}, toolCardStyle(palette, chrome.Color(palette.FG), chrome.Color(palette.Muted), chrome.Color(palette.Border)), ToolImageGalleryOptions{Cwd: "/work", ID: "image-view", Expanded: &expanded, TextOnly: textOnly, OnOpen: func(path string) { opened = append(opened, path) }, OnOpenFile: func(path string) { files = append(files, path) }})
			ui.Box(c).Height(1).Label("Following image flow")
		})
	}, width, 800)
	tt.SetTime(time.Unix(12000, 0))
	tt.Move(float32(width-5), 795)
	return tt, &expanded, &opened, &files
}

func openImageView(t *testing.T, tt *ui.Tester, expanded *map[string]bool) {
	t.Helper()
	(*expanded)["image-view|tool:image-122"] = true
	tt.SetTime(time.Unix(12000, 0).Add(250 * time.Millisecond))
	tt.Frame()
	tt.SetTime(time.Unix(12000, 0).Add(500 * time.Millisecond))
	tt.Frame()
}

func toolImageViewSetLoad(t *testing.T, id, path string, err error) {
	t.Helper()
	attachmentLoads.mu.Lock()
	if attachmentLoads.entries == nil {
		attachmentLoads.entries = map[string]*attachmentLoad{}
	}
	attachmentLoads.entries[id] = &attachmentLoad{path: path, err: err}
	attachmentLoads.mu.Unlock()
}

func toolImageViewBitmap(t *testing.T, width, height int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			ink := color.RGBA{32, 132, 180, 255}
			if x > width/2 || y > height/2 {
				ink = color.RGBA{240, 80, 40, 255}
			}
			img.SetRGBA(x, y, ink)
		}
	}
	var raw bytes.Buffer
	if err := png.Encode(&raw, img); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "recorded-raster.png")
	if err := os.WriteFile(path, raw.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func toolImageViewLoader(t *testing.T) {
	t.Helper()
	SetAttachmentFetcher(func(context.Context, string) (string, error) { return "", errors.New("unexpected fetch") })
	t.Cleanup(func() { SetAttachmentFetcher(nil) })
}

func TestToolImageViewSingleCardMetadataAndIndependentPath(t *testing.T) {
	toolImageViewLoader(t)
	path := toolImageViewBitmap(t, 480, 240)
	for _, dark := range []bool{false, true} {
		t.Run(map[bool]string{false: "light", true: "dark"}[dark], func(t *testing.T) {
			toolImageViewSetLoad(t, "fixture-id-a", path, nil)
			extra := "  first line\nsecond line with `code` and **literal**"
			call := toolImageViewCall(t, false, 1, &extra)
			tt, expanded, opened, files := toolImageViewFixture(t, call, dark, false, 640)
			if !tt.HasText("Image (1)") || tt.HasText("480x240px") {
				t.Fatalf("collapsed suffix/body %v", tt.Texts())
			}
			if err := tt.Click("argument.png"); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(*files, []string{"/work/recorded.png"}) || len(*expanded) != 0 {
				t.Fatalf("header path %v %v", *files, *expanded)
			}
			openImageView(t, tt, expanded)
			for _, want := range []string{"/work/recorded.png", "480x240px (downscaled from 960x480px), 2048 bytes", extra} {
				if !tt.HasText(want) {
					t.Fatalf("missing %q: %v", want, tt.Texts())
				}
			}
			for _, absent := range []string{"IN", "OUT", "Copy", "<path>", "attachmentId", "Images ("} {
				if tt.HasText(absent) {
					t.Fatalf("duplicate %q: %v", absent, tt.Texts())
				}
			}
			frame, ok := tt.Find("sample-a, click to view original")
			if !ok {
				t.Fatal("frame missing")
			}
			if frame.X != 24 || frame.W != 240 || frame.H != 120 {
				t.Fatal("single geometry", frame)
			}
			if err := tt.Click("sample-a, click to view original"); err != nil {
				t.Fatal(err)
			}
			tt.Key(0, ui.KeyEnter)
			tt.Key(0, ui.KeySpace)
			if !reflect.DeepEqual(*opened, []string{path, path, path}) || !(*expanded)["image-view|tool:image-122"] {
				t.Fatal("nested thumbnail activation", *opened, *expanded)
			}
			toolImageExport(t, tt, "single-"+map[bool]string{false: "light", true: "dark"}[dark])
		})
	}
}

func TestToolImageViewTilesWrapAndKeepDuplicates(t *testing.T) {
	toolImageViewLoader(t)
	path := toolImageViewBitmap(t, 480, 240)
	for _, id := range []string{"fixture-id-a", "fixture-id-b", "fixture-id-c"} {
		toolImageViewSetLoad(t, id, path, nil)
	}
	tt, expanded, _, _ := toolImageViewFixture(t, toolImageViewCall(t, true, 3, nil), false, false, 220)
	openImageView(t, tt, expanded)
	frames := []ui.Rect{}
	for _, name := range []string{"sample-a", "sample-b", "sample-c"} {
		r, ok := tt.Find(name + ", click to view original")
		if !ok || r.W != 64 || r.H != 64 {
			t.Fatalf("tile %s %v", name, r)
		}
		frames = append(frames, r)
	}
	if frames[1].X-frames[0].X != 74 || frames[2].X != frames[0].X || frames[2].Y-frames[0].Y != 74 {
		t.Fatal("tile wrap/gap", frames)
	}
	if tt.HasText("/work/recorded.png") || tt.HasText("IN") || tt.HasText("OUT") {
		t.Fatal("generic image body", tt.Texts())
	}
	toolImageExport(t, tt, "tiles-narrow")
}

func TestToolImageViewDeclinedAndUnavailableKeepEvidence(t *testing.T) {
	toolImageViewLoader(t)
	for _, mode := range []string{"no-loader", "invalid-image", "unknown-block", "text-only", "error"} {
		t.Run(mode, func(t *testing.T) {
			toolImageViewLoader(t)
			call := toolImageViewCall(t, false, 1, nil)
			switch mode {
			case "no-loader":
				SetAttachmentFetcher(nil)
			case "invalid-image":
				call.Result.Raw = bytes.ReplaceAll(call.Result.Raw, []byte(`"width":480`), []byte(`"width":0`))
			case "unknown-block":
				call.Result.Raw = bytes.Replace(call.Result.Raw, []byte(`"content":[`), []byte(`"content":[{"type":"extension","value":42},`), 1)
			case "error":
				call.State = "error"
			}
			tt, expanded, opened, files := toolImageViewFixture(t, call, false, mode == "text-only", 640)
			openImageView(t, tt, expanded)
			if !tt.HasText("attachmentId") || !tt.HasText("<path>") || tt.HasText("Image (1)") {
				t.Fatalf("lost evidence %s %v", mode, tt.Texts())
			}
			if mode != "text-only" && tt.HasText("IN") {
				t.Fatal("read-image exposed args")
			}
			if len(*opened)+len(*files) != 0 {
				t.Fatal("fallback opened host")
			}
		})
	}
}

func TestToolImageViewLoadingRetryAndExactID(t *testing.T) {
	path := toolImageViewBitmap(t, 480, 240)
	var requests atomic.Int32
	id := " fixture-id-a "
	entered, release, done := make(chan string, 2), make(chan struct{}), make(chan struct{}, 2)
	SetAttachmentFetcher(func(_ context.Context, actual string) (string, error) {
		n := requests.Add(1)
		entered <- actual
		if n == 1 {
			<-release
			done <- struct{}{}
			return "", errors.New("recorded failure")
		}
		done <- struct{}{}
		return path, nil
	})
	t.Cleanup(func() { SetAttachmentFetcher(nil) })
	call := toolImageViewCall(t, false, 1, nil)
	call.Result.Raw = bytes.ReplaceAll(call.Result.Raw, []byte(`fixture-id-a`), []byte(id))
	tt, expanded, opened, _ := toolImageViewFixture(t, call, false, false, 640)
	openImageView(t, tt, expanded)
	select {
	case actual := <-entered:
		if actual != id {
			t.Fatalf("rewritten %q", actual)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no authorized load")
	}
	if !tt.HasText("Loading image…") {
		t.Fatal("loading placeholder", tt.Texts())
	}
	frame, _ := tt.Find("sample-a, click to view original")
	if frame.W != 240 || frame.H != 120 {
		t.Fatal("loading fit", frame)
	}
	_ = tt.Click("sample-a, click to view original")
	if len(*opened) != 0 {
		t.Fatal("pending image opened")
	}
	close(release)
	<-done
	toolImageWait(t, tt, func() bool { return tt.HasText("Image failed to load; click to retry") })
	tt.Move(630, 790)
	tt.Move(629, 790)
	if requests.Load() != 1 {
		t.Fatal("automatic retry")
	}
	if err := tt.Click("Image failed to load; click to retry"); err != nil {
		t.Fatal(err)
	}
	select {
	case actual := <-entered:
		if actual != id {
			t.Fatal(actual)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("retry did not reach loader")
	}
	<-done
	toolImageWait(t, tt, func() bool {
		return !tt.HasText("Loading image…") && !tt.HasText("Image failed to load; click to retry")
	})
	if requests.Load() != 2 {
		t.Fatal("retry duplicated", requests.Load())
	}
	if err := tt.Click("sample-a, click to view original"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*opened, []string{path}) || !(*expanded)["image-view|tool:image-122"] {
		t.Fatal(*opened, *expanded)
	}
}

func toolImageWait(t *testing.T, tt *ui.Tester, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		tt.Move(631, 790)
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("image state did not settle", tt.Texts())
}

func TestToolImageViewSizingCropAndLocales(t *testing.T) {
	for _, tc := range []struct {
		w, h         float64
		wantW, wantH float32
	}{{480, 240, 240, 120}, {240, 480, 120, 240}, {32, 16, 44, 44}, {2, 3, 44, 44}, {100, 2000, 60, 240}, {2000, 100, 240, 60}, {1e300, 1, 44, 44}, {1, 1e300, 44, 44}} {
		w, h := toolImageFit(tc.w, tc.h, false)
		if w != tc.wantW || h != tc.wantH {
			t.Fatal(tc, w, h)
		}
	}
	bitmap := loadChatBitmap(toolImageViewBitmap(t, 100, 2000))
	box := ui.Rect{X: 10, Y: 20, W: 60, H: 240}
	crop := toolImageCrop(bitmap, box, .05, false)
	if crop.Y != box.Y || crop.H <= box.H {
		t.Fatal("tall top anchor", crop)
	}
	crop = toolImageCrop(bitmap, box, .05, true)
	if crop.Y >= box.Y {
		t.Fatal("tile center anchor", crop)
	}
	old := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(old) })
	for _, locale := range []string{i18n.LocaleEN, i18n.LocaleZH} {
		i18n.SetLocale(locale)
		text := toolImageSizeText(toolImageAttachment{Width: 1e21, Height: 240, Bytes: 2048})
		if !strings.Contains(text, "1e+21x240px") || strings.Contains(text, "2,048") {
			t.Fatal(text)
		}
	}
	if !toolUsesGenericImageCard("render_plot") || !toolUsesGenericImageCard("cordis_run") || toolUsesGenericImageCard("read_image") || toolUsesGenericImageCard("lsp") {
		t.Fatal("keyed image priority")
	}
}

func toolImageExport(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("TOOL_IMAGE_EVIDENCE_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "tool-image-pass266-native-software-"+name)
	file, err := os.Create(path + ".png")
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, tt.Image()); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	boxes := map[string]ui.Rect{}
	for _, label := range []string{"sample-a, click to view original", "sample-b, click to view original", "sample-c, click to view original", "/work/recorded.png", "480x240px (downscaled from 960x480px), 2048 bytes", "Following image flow"} {
		if rect, ok := tt.Find(label); ok {
			boxes[label] = rect
		}
	}
	raw, _ := json.MarshalIndent(map[string]any{"boxes": boxes, "texts": tt.Texts()}, "", "  ")
	if err := os.WriteFile(path+".json", raw, 0600); err != nil {
		t.Fatal(err)
	}
}
