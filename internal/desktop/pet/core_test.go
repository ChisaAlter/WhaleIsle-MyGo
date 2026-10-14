package pet

import (
	"encoding/binary"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"whaleisle/internal/config"
)

// This single fixture covers the offline control path and image-only boundary;
// actual window appearance/dragging remains a Windows interactive gate.
func TestDesktopPetCore(t *testing.T) {
	dir := t.TempDir()
	original := []byte(`{"pet":{"enabled":true},"live2dPet":{"enabled":true,"growth":{"points":7}},"unknown":"keep"}`)
	if err := os.WriteFile(filepath.Join(dir, config.FileName), original, 0600); err != nil {
		t.Fatal(err)
	}
	store, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if store.DesktopPet().Enabled {
		t.Fatal("must default off")
	}
	if err = store.SaveDesktopPet(config.DesktopPet{Enabled: true, SkinID: "example"}); err != nil {
		t.Fatal(err)
	}
	again, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !again.DesktopPet().Enabled || again.DesktopPet().SkinID != "example" {
		t.Fatal("selection not restored")
	}
	data, _ := os.ReadFile(filepath.Join(dir, config.FileName))
	var obj map[string]json.RawMessage
	_ = json.Unmarshal(data, &obj)
	if string(obj["unknown"]) != `"keep"` || len(obj["live2dPet"]) == 0 || len(obj["pet"]) == 0 {
		t.Fatal("legacy configuration changed")
	}
	if err = again.Update(func(c *config.Config) { c.Port = 4001 }); err != nil {
		t.Fatal(err)
	}
	if err = store.SaveDesktopPet(config.DesktopPet{}); err != config.ErrConflict {
		t.Fatalf("stale store write: %v", err)
	}
	root := filepath.Join(dir, "pets")
	_ = os.MkdirAll(filepath.Join(root, "example"), 0700)
	sheet := make([]byte, 30)
	copy(sheet, "RIFF")
	binary.LittleEndian.PutUint32(sheet[4:], 22)
	copy(sheet[8:], "WEBPVP8X")
	binary.LittleEndian.PutUint32(sheet[16:], 10)
	sheet[24] = 255
	sheet[25] = 5
	sheet[27] = 239
	sheet[28] = 8 // 1536 x 2288
	_ = os.WriteFile(filepath.Join(root, "example", "sheet.webp"), sheet, 0600)
	_ = os.WriteFile(filepath.Join(root, "example", "pet.json"), []byte(`{"spritesheetPath":"sheet.webp","displayName":"Example"}`), 0600)
	skins, err := Discover(root)
	if err != nil || len(skins) != 1 || skins[0].Rows != 11 {
		t.Fatalf("catalog: %+v, %v", skins, err)
	}
	_ = os.MkdirAll(filepath.Join(root, "escape"), 0700)
	_ = os.WriteFile(filepath.Join(root, "escape", "pet.json"), []byte(`{"spritesheetPath":"../example/sheet.webp"}`), 0600)
	_ = os.MkdirAll(filepath.Join(root, "link"), 0700)
	_ = os.WriteFile(filepath.Join(root, "link", "pet.json"), []byte(`{"spritesheetPath":"sheet.webp"}`), 0600)
	_ = os.Symlink(filepath.Join(root, "example", "sheet.webp"), filepath.Join(root, "link", "sheet.webp"))
	skins, err = Discover(root)
	if err != nil || len(skins) != 1 {
		t.Fatalf("unsafe package admitted: %+v %v", skins, err)
	}
	p := &Companion{sheet: skins[0].Sheet}
	for _, route := range []struct {
		url  string
		code int
	}{{"whale-pet://localhost/index.html", 200}, {"whale-pet://localhost/sprite.webp", 200}, {"whale-pet://localhost/fallback.png", 200}, {"whale-pet://localhost/../config.json", 404}, {"whale-pet://wrong/index.html", 404}} {
		req := httptest.NewRequest("GET", route.url, nil)
		res := httptest.NewRecorder()
		p.ServeHTTP(res, req)
		if res.Code != route.code {
			t.Fatalf("%s: %d", route.url, res.Code)
		}
	}
}
