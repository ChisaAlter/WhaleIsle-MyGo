package session

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAgentNavigationImageContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "photo.png")
	data := []byte("\x89PNG\r\n\x1a\nimage fixture")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	parts, err := agentNavigationPromptContent("look", []string{path})
	if err != nil || len(parts) != 2 {
		t.Fatalf("mixed prompt: %s %v", parts, err)
	}
	var image map[string]string
	if err := json.Unmarshal(parts[1], &image); err != nil {
		t.Fatal(err)
	}
	if image["type"] != "image" || image["mediaType"] != "image/png" || image["data"] != base64.StdEncoding.EncodeToString(data) {
		t.Fatalf("image wire: %v", image)
	}
	if parts, err = agentNavigationPromptContent("", []string{path}); err != nil || len(parts) != 1 {
		t.Fatalf("image-only: %s %v", parts, err)
	}
	bad := filepath.Join(t.TempDir(), "pretend.png")
	_ = os.WriteFile(bad, []byte("not an image"), 0600)
	if _, err = agentNavigationPromptContent("", []string{bad}); err == nil {
		t.Fatal("file admitted as image")
	}
}
