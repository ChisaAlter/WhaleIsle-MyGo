package pet

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
)

var realPetEvidence = map[string]any{}

// GUI validation is explicit and runs in its own test executable. Compile with
// -ldflags=-H=windowsgui and start via scripts/run-hidden.ps1. Every OS window
// belongs to this process and every persisted file lives in a fresh temp root.
// No existing app, lock, profile, process or user cursor is changed by this test.
func TestMain(m *testing.M) {
	if os.Getenv("WHALEISLE_PET_GUI") != "1" {
		os.Exit(m.Run())
	}
	dir, err := os.MkdirTemp("", "whaleisle-pet-gui-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	realPetEvidence["pid"] = os.Getpid()
	executable, _ := os.Executable()
	binary, _ := os.ReadFile(executable)
	sum := sha256.Sum256(binary)
	realPetEvidence["executable"] = executable
	realPetEvidence["executableSHA256"] = hex.EncodeToString(sum[:])
	realPetEvidence["profile"] = dir
	mygo.App.SetName("WhaleIsle Pet Isolated Check")
	mygo.App.SetPath(mygo.PathUserData, filepath.Join(dir, "profile"))
	mygo.App.SetPath(mygo.PathCache, filepath.Join(dir, "cache"))
	_ = os.Setenv("CODEX_HOME", filepath.Join(dir, "codex"))
	mygo.App.OnWindowAllClosed(func() {})
	code := 1
	mygo.App.WhenReady(func() { go func() { code = m.Run(); mygo.App.Quit() }() })
	if err := mygo.App.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		code = 1
	}
	// WebView2 may still hold the owned directory while shutting down. A failed
	// cleanup is reported rather than reaching out to terminate any process.
	if err := os.RemoveAll(dir); err != nil {
		fmt.Fprintln(os.Stderr, "owned GUI temp cleanup:", err)
	}
	realPetEvidence["exitCode"] = code
	realPetEvidence["pass"] = code == 0 && realPetEvidence["dom"] != nil
	if result := os.Getenv("WHALEISLE_PET_GUI_RESULT"); result != "" {
		data, _ := json.MarshalIndent(realPetEvidence, "", "  ")
		if err := os.WriteFile(result, data, 0600); err != nil {
			code = 1
		}
	}
	os.Exit(code)
}

func savePetGUIImage(t *testing.T, name string, data []byte) {
	t.Helper()
	result := os.Getenv("WHALEISLE_PET_GUI_RESULT")
	if result == "" {
		return
	}
	path := filepath.Join(filepath.Dir(result), "pet-"+name+".png")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Error(err)
		return
	}
	realPetEvidence[name+"Screenshot"] = path
}

func TestRealPetOverlayAndQuickChatCreation(t *testing.T) {
	if runtime.GOOS != "windows" || os.Getenv("WHALEISLE_PET_GUI") != "1" {
		t.Skip("explicit Windows GUI fixture only")
	}
	store, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	neural := os.Getenv("WHALEISLE_PET_GUI_NEURAL") == "1"
	if neural {
		dir := os.Getenv("WHALEISLE_PET_RENDER_ASSETS")
		if dir == "" {
			t.Fatal("neural GUI fixture requires staged resources")
		}
		previous := petRenderRoot
		petRenderRoot = func() string { return dir }
		t.Cleanup(func() { petRenderRoot = previous })
	}
	if err := store.SaveDesktopPet(config.DesktopPet{Enabled: true, Neural: neural, Behavior: config.DefaultPetBehavior()}); err != nil {
		t.Fatal(err)
	}
	var p *Companion
	var w *mygo.Window
	mygo.RunOnMain(func() { p = New(store); p.SetReady(true); w = p.win })
	if w == nil || w.IsDestroyed() {
		t.Fatal("real companion creation failed")
	}
	t.Cleanup(func() { mygo.RunOnMain(p.Close) })
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	var observed any
	for ctx.Err() == nil {
		observed, err = w.Page().EvalContext(ctx, `document.body.dataset.renderer && (()=>{const r=document.getElementById("sprite").getBoundingClientRect();return {width:r.width,height:r.height,transparent:getComputedStyle(document.body).backgroundColor,renderer:document.body.dataset.renderer,provider:document.body.dataset.provider||"",gpuInputs:document.body.dataset.gpuInputs||"",graphCapture:document.body.dataset.graphCapture||"",superResolution:document.body.dataset.superResolution||"",webgpuSecure:window.isSecureContext,webgpuAvailable:!!navigator.gpu};})()`)
		if err == nil && observed != nil && observed != false {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil || observed == nil || observed == false {
		t.Fatalf("real renderer did not paint: %v %v", observed, err)
	}
	value, ok := observed.(map[string]any)
	if !ok || value["width"] != float64(240) || value["height"] != float64(260) || value["transparent"] != "rgba(0, 0, 0, 0)" {
		t.Fatalf("actual DOM geometry: %#v", observed)
	}
	if neural && (value["renderer"] != "neural" || value["superResolution"] != "anime4k") {
		t.Errorf("real neural/SR first frame did not succeed: %#v", observed)
	}
	realPetEvidence["dom"] = observed
	var before, after mygo.Rectangle
	var chatWindow *mygo.Window
	mygo.RunOnMain(func() {
		before = petWindowBounds(w)
		want := displayOverlayBounds(petDisplays(), p.petBounds())
		if before != want {
			t.Errorf("real overlay bounds=%+v want=%+v", before, want)
		}
		p.setMotionPosition(motionPoint{x: 50, y: 80})
		after = petWindowBounds(w)
		if after != before || p.petPosition() != (motionPoint{x: 50, y: 80}) {
			t.Errorf("logical movement moved overlay: %+v -> %+v stage=%+v", before, after, p.stage)
		}
		previousOpen, previousFocus := openQuickChatWindow, focusQuickChatWindow
		focusQuickChatWindow = func(*mygo.Window) {}
		openQuickChatWindow = func(opts mygo.WindowOptions) *mygo.Window {
			opts.Hidden = true
			card := mygo.NewWindow(opts)
			card.OnReadyToShow(card.ShowInactive)
			return card
		}
		p.OpenQuickChat(QuickChatOptions{Unavailable: "isolated fixture offline"})
		openQuickChatWindow, focusQuickChatWindow = previousOpen, previousFocus
		if p.QuickChat == nil || p.QuickChat.win == nil || p.QuickChat.win.IsDestroyed() {
			t.Error("real compact card creation failed")
			return
		}
		chatWindow = p.QuickChat.win
		bounds := chatWindow.Bounds()
		realPetEvidence["overlayBounds"] = before
		realPetEvidence["nativeScale"] = petWindowScale(w)
		realPetEvidence["overlayDIPBounds"] = w.Bounds()
		realPetEvidence["quickChatBounds"] = bounds
		realPetEvidence["logicalStage"] = p.stage
		if bounds.Width != 248 || bounds.Height != 360 {
			t.Errorf("real compact card bounds=%+v", bounds)
		}
		// Exercise the production pass-through transition without moving the
		// user cursor or synthesizing desktop clicks.
		p.interaction.hit.report = petHitReport{Width: float64(before.Width), Height: float64(before.Height), Regions: []petHitRect{{X: 40, Y: 40, Width: 100, Height: 100}}}
		p.updateHitRegions(mygo.Point{X: before.X + 500, Y: before.Y + 500})
		if !p.interaction.hit.ignoring {
			t.Error("overlay blank did not request mouse passthrough")
		}
		realPetMouseStyle(t, w, true)
		p.updateHitRegions(mygo.Point{X: before.X + 80, Y: before.Y + 80})
		if p.interaction.hit.ignoring {
			t.Error("visible island stayed click-through")
		}
		realPetMouseStyle(t, w, false)
	})
	if chatWindow != nil {
		image, err := chatWindow.CapturePage()
		if err != nil {
			t.Errorf("own chat capture: %v", err)
		} else {
			savePetGUIImage(t, "quickchat", image)
		}
	}
	if image, err := w.CapturePage(); err == nil {
		savePetGUIImage(t, "overlay", image)
	} else {
		t.Errorf("own pet capture: %v", err)
	}
	mygo.RunOnMain(func() {
		p.CloseQuickChat()
		cfg := p.Store.DesktopPet()
		cfg.Behavior.Scale = 1.6
		if err := p.Store.SaveDesktopPet(cfg); err != nil {
			t.Error(err)
			return
		}
		p.applyBehavior()
	})
	var scaled any
	for ctx.Err() == nil {
		scaled, err = w.Page().EvalContext(ctx, `(()=>{const r=document.getElementById("sprite").getBoundingClientRect();return {width:r.width,height:r.height,left:r.left,top:r.top};})()`)
		if v, ok := scaled.(map[string]any); ok && v["width"] == float64(384) && v["height"] == float64(416) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if v, ok := scaled.(map[string]any); !ok || v["width"] != float64(384) || v["height"] != float64(416) {
		t.Errorf("real scaled CSS stage=%v err=%v", scaled, err)
	}
	realPetEvidence["scaledDOM"] = scaled
	t.Logf("real WebView2 %v; compact 248x360; overlay logical motion/click-through transitions checked", observed)
}
