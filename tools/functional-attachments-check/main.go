// Isolated CHAT-10 acceptance. Creates only owned idle sessions and files;
// never submits a prompt, invokes a model, opens a picker or reads clipboard.
package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

type report struct {
	Revision, Root                    string
	ModelRequests                     int
	Pass, HostStopped, ListenerClosed bool
	Checks                            map[string]any
	Error                             string `json:",omitempty"`
}

func main() {
	stage := flag.String("stage", "", "verified runtime")
	root := flag.String("root", "", "new isolated absolute directory")
	output := flag.String("out", "", "JSON evidence")
	visualOnly := flag.Bool("visual-only", false, "only actual native capture using explicitly marked local visual fixture")
	flag.Parse()
	r := report{Root: *root, Checks: map[string]any{}}
	if v, e := exec.Command("git", "rev-parse", "HEAD").Output(); e == nil {
		r.Revision = strings.TrimSpace(string(v))
	}
	if !filepath.IsAbs(*root) || !filepath.IsAbs(*stage) || !filepath.IsAbs(*output) {
		panic("absolute paths required")
	}
	if e := os.Mkdir(*root, 0700); e != nil {
		panic(e)
	}
	for k, v := range map[string]string{paths.EnvUserData: filepath.Join(*root, "data"), paths.EnvVendor: filepath.Join(*stage, "vendor"), "DSH_HOME": filepath.Join(*root, "data", "dsh-home")} {
		if e := os.Setenv(k, v); e != nil {
			panic(e)
		}
	}
	mygo.App.SetName("Owned attachment acceptance")
	mygo.App.SetPath(mygo.PathUserData, filepath.Join(*root, "native"))
	mygo.App.SetPath(mygo.PathCache, filepath.Join(*root, "cache"))
	mygo.App.SetPath(mygo.PathLogs, filepath.Join(*root, "logs"))
	// Cleanup and evidence must finish after the last owned window is destroyed.
	mygo.App.OnWindowAllClosed(func() {})
	mygo.App.WhenReady(func() {
		go func() {
			defer mygo.App.Quit()
			if e := run(*stage, *root, *visualOnly, &r); e != nil {
				r.Error = e.Error()
			} else {
				r.Pass = true
			}
			b, e := json.MarshalIndent(r, "", "  ")
			if e == nil {
				e = os.WriteFile(*output, append(b, '\n'), 0600)
			}
			if e != nil {
				r.Pass = false
				fmt.Fprintln(os.Stderr, e)
			}
			fmt.Println(string(b))
		}()
	})
	mygo.App.Run()
	if !r.Pass {
		os.Exit(1)
	}
}

func run(stage, root string, visualOnly bool, r *report) error {
	workspace := filepath.Join(root, "workspace")
	if e := os.Mkdir(workspace, 0700); e != nil {
		return e
	}
	text := []byte("owned attachment 中文\nnot a model prompt\n")
	path := filepath.Join(workspace, "note.txt")
	if e := os.WriteFile(path, text, 0600); e != nil {
		return e
	}
	picture := image.NewRGBA(image.Rect(0, 0, 80, 48))
	for y := 0; y < 48; y++ {
		for x := 0; x < 80; x++ {
			picture.Set(x, y, color.RGBA{R: uint8(80 + x), B: 200, A: 255})
		}
	}
	var pb bytes.Buffer
	if e := png.Encode(&pb, picture); e != nil {
		return e
	}
	imagePath := filepath.Join(workspace, "photo.png")
	if e := os.WriteFile(imagePath, pb.Bytes(), 0600); e != nil {
		return e
	}
	if visualOnly {
		d := composer.NewDraftState("owned unsent draft")
		d.Attachments = []session.ComposerAttachment{
			{Path: path, Name: "note.txt", Upload: api.FileUploadValue{ReceiptId: "visual-fixture", File: api.FileUploadValueFile{Bytes: float64(len(text))}}},
			{Path: imagePath, Name: "photo.png", ImageData: base64.StdEncoding.EncodeToString(pb.Bytes())},
		}
		r.Checks["scope"] = "actual native UI only; local visual fixture for text-ready metadata and PNG; no Host started"
		return capture(root, r, d)
	}
	vendor := filepath.Join(stage, "vendor")
	hr := filepath.Join(vendor, "deepseek-harness")
	node := filepath.Join(stage, "node", "node.exe")
	home := filepath.Join(root, "data", "dsh-home")
	sup := harness.New(harness.Options{Install: harness.NewInstallServer(harness.InstallServerOptions{NodeBin: node}), Launch: harness.LaunchOptions{NodeBin: node, HarnessRoot: hr, VendorDir: vendor, Host: "127.0.0.1", Port: 32379, Home: home}, Prepare: harness.PrepareOptions{VendorDir: vendor, HarnessRoot: hr, DshHome: home, ProfileDir: filepath.Join(home, "profiles", "web"), RemoteSet: true, Remote: false}, Workspace: workspace, ReadyTimeout: 3 * time.Minute})
	var ep harness.Endpoint
	defer func() {
		sup.Stop()
		r.HostStopped = sup.Status().State == harness.Stopped
		if u, e := url.Parse(ep.Origin); e == nil && u.Host != "" {
			c, e := net.DialTimeout("tcp", u.Host, time.Second)
			r.ListenerClosed = e != nil
			if c != nil {
				c.Close()
			}
		}
	}()
	sup.Start()
	ctx, cancel := context.WithTimeout(context.Background(), 195*time.Second)
	defer cancel()
	for {
		v := sup.Status()
		if v.State == harness.Failed {
			return v.Err
		}
		if v.State == harness.Ready {
			ep = v.Endpoint
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: ep.Origin, Cookie: ep.Cookie}, HTTP: &http.Client{Timeout: 20 * time.Second}}
	store := session.New()
	store.Attach(client, nil)
	defer store.Detach()
	create := func() (string, error) {
		var v api.SessionCreateValue
		e := client.Call(ctx, "session/create", map[string]any{"request": api.SessionCreateRequest{Cwd: &workspace}}, &v)
		if e == nil && v.SessionId == "" {
			e = errors.New("empty owned session")
		}
		return v.SessionId, e
	}
	a, e := create()
	if e != nil {
		return e
	}
	b, e := create()
	if e != nil {
		return e
	}
	d := composer.NewDraftState("owned unsent draft")
	d.Attachments = []session.ComposerAttachment{{Path: path, Name: "note.txt"}, {Path: imagePath, Name: "photo.png"}}
	prepare := func(index int, retry bool, owner string) error {
		before, ok := d.BeginAttachmentPreparation(index, retry)
		if !ok {
			return errors.New("attachment preparation was not admitted")
		}
		if _, duplicate := d.BeginAttachmentPreparation(index, retry); duplicate {
			return errors.New("duplicate preparation admitted")
		}
		target, e := store.CaptureComposerTarget(owner)
		if e != nil {
			return e
		}
		after, e := target.PrepareAttachment(ctx, before)
		d.CompleteAttachmentPreparation(index, before, after, e)
		return e
	}
	if e = prepare(0, false, a); e != nil {
		return e
	}
	if e = prepare(1, false, a); e != nil {
		return e
	}
	first := d.Attachments[0]
	if first.Upload.ReceiptId == "" || first.Upload.File.Bytes != float64(len(text)) || d.Attachments[1].ImageData != base64.StdEncoding.EncodeToString(pb.Bytes()) {
		return errors.New("text receipt/image preparation mismatch")
	}
	r.Checks["realTextUploadAndPNGPreparation"] = "pass; receipt bytes equal source; PNG exact base64; repeated click refused while pending"
	cached, e := store.PrepareComposerAttachment(ctx, a, first)
	if e != nil || cached.Upload.ReceiptId != first.Upload.ReceiptId {
		return errors.New("same-owner receipt was not reused")
	}
	other, e := store.PrepareComposerAttachment(ctx, b, first)
	if e != nil || other.SessionID != b || other.Upload.ReceiptId == first.Upload.ReceiptId {
		return errors.New("session switch reused prior receipt")
	}
	r.Checks["sessionOwnerIsolation"] = "pass; same owner reused receipt, second idle session obtained a distinct fresh receipt"
	session.SetAttachmentCacheDir(filepath.Join(root, "attachment-cache"))
	store.Select(a)
	fetched, e := store.FetchAttachment(ctx, first.Upload.File.AttachmentId)
	if e != nil {
		r.Checks["stagedFetch"] = "Host refused unattached staged blob before prompt: " + e.Error()
	} else {
		raw, e := os.ReadFile(fetched)
		if e != nil || !bytes.Equal(raw, text) {
			return errors.New("fetched attachment bytes differ")
		}
		r.Checks["stagedFetch"] = "pass; real session/attachment bytes equal source"
	}
	restored, e := composer.RestoreDraftChecked(d.SaveDraft())
	if e != nil {
		return e
	}
	if restored.Text != d.Text || len(restored.Attachments) != 2 || restored.Attachments[0].Upload.ReceiptId != "" || restored.Attachments[1].ImageData != "" {
		return errors.New("recovery persisted Host-owned staging data")
	}
	recovered, e := store.PrepareComposerAttachment(ctx, a, restored.Attachments[0])
	if e != nil || recovered.Upload.ReceiptId == "" || recovered.Upload.ReceiptId == first.Upload.ReceiptId {
		return errors.New("restored path was not freshly prepared")
	}
	r.Checks["durableDraftRecovery"] = "pass; Path/Name/Text survive actual JSON roundtrip, receipt/image bytes excluded and fresh receipt acquired"
	missing := filepath.Join(workspace, "retry.txt")
	d.Attachments = append(d.Attachments, session.ComposerAttachment{Path: missing, Name: "retry.txt"})
	if e = prepare(2, false, a); e == nil || d.Attachments[2].PrepareError == "" || d.Text != "owned unsent draft" {
		return errors.New("missing file failure did not retain draft")
	}
	if e = os.WriteFile(missing, []byte("retry succeeds"), 0600); e != nil {
		return e
	}
	if e = prepare(2, true, a); e != nil {
		return e
	}
	if d.Preparing != 0 || d.Attachments[2].PrepareError != "" {
		return errors.New("retry did not settle")
	}
	r.Checks["missingFileRetry"] = "pass; local read failure retained file and text; file creation then explicit retry completed"
	if !d.RemoveAttachment(2) || len(d.Attachments) != 2 {
		return errors.New("remove failed")
	}
	if _, e = os.Stat(missing); e != nil {
		return errors.New("attachment remove changed source file")
	}
	r.Checks["remove"] = "pass; draft reference removed, actual source file unchanged; Host staged receipts are not remotely deleted"
	r.Checks["notPerformed"] = []string{"OS clipboard", "native picker/drop pointer operation", "prompt submission/consumption of staged receipt", "uploaded attachment history without prompt"}
	return capture(root, r, d)
}

func capture(root string, r *report, d *composer.DraftState) error {
	model := composer.Model{State: d, Draft: &d.Text, UseTextInputClient: true, TextInput: d.Client, ThemeID: "light"}
	headless := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(20).Children(func() { composer.View(c, model) })
	}, 800, 400)
	nameRect, nameFound := headless.Find("note.txt")
	cardRect, cardFound := headless.Find("Attachment note.txt")
	r.Checks["composerGeometry"] = map[string]any{"name": nameRect, "nameFound": nameFound, "card": cardRect, "cardFound": cardFound}
	if !nameFound || nameRect.W < 100 || nameRect.H != 22 || !cardFound || cardRect.H != 64 || nameRect.Y < cardRect.Y || nameRect.Y+nameRect.H > cardRect.Y+cardRect.H {
		return fmt.Errorf("full composer compact file name geometry: name=%+v card=%+v", nameRect, cardRect)
	}
	win := mygo.NewWindow(mygo.WindowOptions{Title: "Owned attachment composer", Width: 800, Height: 400, Hidden: true, SkipTaskbar: true, Content: ui.View(func(c *ui.Context) { ui.Column(c).Fill().Padding(20).Children(func() { composer.View(c, model) }) })})
	if win == nil {
		return errors.New("owned native composer window unavailable")
	}
	defer win.Destroy()
	// Hidden windows retain their initial captured scene. Show only this owned
	// window without activation so the real message loop can present the
	// metadata, decoded thumbnail and asynchronously prepared font faces.
	win.ShowInactive()
	if _, e := win.CapturePage(); e != nil {
		return e
	}
	time.Sleep(400 * time.Millisecond)
	shot, e := win.CapturePage()
	if e != nil {
		return e
	}
	if e = os.WriteFile(filepath.Join(root, "composer-attachments.png"), shot, 0600); e != nil {
		return e
	}
	r.Checks["nativeScreenshot"] = filepath.Join(root, "composer-attachments.png")
	return nil
}
