package remote

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/phoneremote"
	remotestore "whaleisle/internal/store/remote"
)

func saveSurfaceSnapshot(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("SURFACES_GALLERY_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, tt.Image()); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

// TestRemoteMachineCardsSnapshot renders the machine list for
// evidence/after/surface-remote-machines.png.
func TestRemoteMachineCardsSnapshot(t *testing.T) {
	snap := remotestore.Snapshot{
		Loaded:    true,
		CurrentID: "a",
		Connected: true,
		Machines: []remotestore.Machine{
			{ID: "a", Name: "构建机", Host: "10.0.0.12", Port: 22, Username: "deploy", LatencyMS: 18, LastConnectedAt: "2026-10-20T16:40:00Z", PasswordSet: true},
			{ID: "b", Name: "测试机", Host: "lab.internal", Port: 2222, Username: "qa", Workspace: "/srv/app"},
			{ID: "c", Name: "", Host: "192.168.1.7", Port: 22, Username: "root"},
		},
	}
	editor := &MachineEditor{}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().MinWidth(0).Padding(16).Gap(10).Children(func() {
			for _, machine := range snap.Machines {
				machineRow(c, Model{}, snap, editor, machine, c.Theme().TextMuted)
			}
		})
	}, 720, 320)
	saveSurfaceSnapshot(t, tt, "surface-remote-machines.png")
}

// TestRemoteFilesSnapshot renders the file page for
// evidence/after/surface-remote-files.png.
func TestRemoteFilesSnapshot(t *testing.T) {
	dir := remotestore.DirectorySnapshot{Loaded: true, Path: "/srv/app/src", Items: []remotestore.DirectoryEntry{
		{Name: "app", Path: "/srv/app/src/app", Type: "dir"},
		{Name: "wailsjs", Path: "/srv/app/src/wailsjs", Type: "dir"},
		{Name: "main.go", Path: "/srv/app/src/main.go", Type: "file"},
		{Name: "go.mod", Path: "/srv/app/src/go.mod", Type: "file"},
	}}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().MinWidth(0).Padding(16).Gap(10).Children(func() {
			directoryBreadcrumb(c, Model{}, dir, false, c.Theme().TextMuted)
			ui.Column(c).FillWidth().Gap(1).Children(func() {
				for _, entry := range dir.Items {
					directoryRow(c, Model{}, entry, false, c.Theme().TextMuted)
				}
			})
		})
	}, 640, 260)
	saveSurfaceSnapshot(t, tt, "surface-remote-files.png")
}

// TestRemotePairingSnapshot renders the pairing steps for
// evidence/after/surface-pairing.png.
func TestRemotePairingSnapshot(t *testing.T) {
	runtime := phoneremote.RuntimeState{
		Status:         "ready",
		Listening:      true,
		RelayConnected: true,
		PairingURL:     "https://connect.example.invalid/pair/abc",
		ExpiresAt:      time.Now().Add(30 * time.Minute),
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().MinWidth(0).Padding(16).Gap(10).Children(func() {
			pairingSteps(c, runtime, "ready", false, c.Theme().Text, c.Theme().TextMuted, c.Theme().Accent)
		})
	}, 720, 160)
	saveSurfaceSnapshot(t, tt, "surface-pairing.png")
}
