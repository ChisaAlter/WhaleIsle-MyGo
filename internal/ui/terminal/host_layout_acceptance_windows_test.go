//go:build windows && terminalacceptance

package terminal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	myterm "github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/hostterminal"
	"whaleisle/internal/ui/theme"
)

// Offline terminal-content replay for layout only: no Host, process or socket.
type recordedLayoutWire struct {
	closed chan struct{}
	once   sync.Once
}

func (w *recordedLayoutWire) Read([]byte) (int, error)    { <-w.closed; return 0, io.EOF }
func (w *recordedLayoutWire) Write(p []byte) (int, error) { return len(p), nil }
func (w *recordedLayoutWire) Close() error                { w.once.Do(func() { close(w.closed) }); return nil }

func TestHostSurfaceRecordedNativeLayout(t *testing.T) {
	out := os.Getenv("WHALEISLE_HOST_LAYOUT_OUTPUT")
	recording := os.Getenv("WHALEISLE_HOST_LAYOUT_RECORDING")
	if out == "" || recording == "" || os.Getenv("MYGO_GHOSTTY_VT") == "" {
		t.Fatal("explicit owned layout output, recording and staged DLL required")
	}
	if err := os.MkdirAll(out, 0700); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(recording)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	type frame struct {
		AttachmentID string              `json:"attachmentId"`
		Render       hostterminal.Render `json:"render"`
	}
	frames := []frame{}
	latest := ""
	scan := bufio.NewScanner(f)
	scan.Buffer(make([]byte, 4096), 8<<20)
	for scan.Scan() {
		var p frame
		if err = json.Unmarshal(scan.Bytes(), &p); err != nil {
			t.Fatal(err)
		}
		frames = append(frames, p)
		if p.Render.Snapshot {
			latest = p.AttachmentID
		}
	}
	if err = scan.Err(); err != nil {
		t.Fatal(err)
	}
	if latest == "" {
		t.Fatal("recorded authoritative snapshot missing")
	}
	results := []map[string]any{}
	for _, size := range [][2]int{{1000, 540}, {800, 420}, {360, 420}, {320, 300}} {
		name := fmt.Sprintf("%dx%d", size[0], size[1])
		t.Run(name, func(t *testing.T) {
			wire := &recordedLayoutWire{closed: make(chan struct{})}
			colors := theme.Resolve(theme.IDDeepseekDark).Active(true)
			themed := terminalTheme(colors, true)
			term, e := myterm.New(myterm.Options{Conn: wire, ReadOnly: true, Theme: themed, DarkTheme: themed, Font: terminalFont(theme.RenderPreferences().Terminal)})
			if e != nil {
				t.Fatal(e)
			}
			for _, p := range frames {
				if p.AttachmentID == latest {
					if p.Render.Snapshot {
						term.Resize(p.Render.Cols, p.Render.Rows)
					}
					term.Feed([]byte(p.Render.Data))
				}
			}
			h := &HostState{terminal: term}
			store := &hostterminal.Store{}
			v := hostLayoutView()
			tt := ui.NewTester(func(c *ui.Context) { hostSurface(c, h, store, "layout-session", colors, v, HostActions{}) }, size[0], size[1])
			tt.SetDark(true)
			tt.Frame()
			body, ok := tt.Find("Terminal")
			if !ok || body.H < float32(size[1]-80) {
				t.Fatalf("%s terminal body=%+v", name, body)
			}
			cols, rows := term.Size()
			if rows < 10 {
				t.Fatalf("%s usable terminal rows=%d", name, rows)
			}
			image := filepath.Join(out, name+".png")
			writeLayoutPNG(t, image, tt)
			result := map[string]any{"viewport": size, "body": body, "cols": cols, "rows": rows, "image": image, "source": "software HostView surface + offline recorded terminal replay; no Host/window/PTy started"}
			if size[0] == 360 {
				if e = tt.Click(terminalLabel("终端管理", "Terminal management")); e != nil {
					t.Fatal(e)
				}
				tt.Frame()
				after, _ := tt.Find("Terminal")
				if after != body {
					t.Fatal("management overlay changed body geometry")
				}
				manageImage := filepath.Join(out, name+"-management.png")
				writeLayoutPNG(t, manageImage, tt)
				result["managementImage"] = manageImage
			}
			results = append(results, result)
			if e = term.Close(); e != nil {
				t.Fatal(e)
			}
		})
	}
	b, _ := json.MarshalIndent(results, "", "  ")
	if err = os.WriteFile(filepath.Join(out, "geometry.json"), append(b, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	t.Log(string(b))
}

func writeLayoutPNG(t *testing.T, path string, tt *ui.Tester) {
	t.Helper()
	file, e := os.Create(path)
	if e != nil {
		t.Fatal(e)
	}
	defer file.Close()
	if e = png.Encode(file, tt.Image()); e != nil {
		t.Fatal(e)
	}
}
