package session

import (
	"encoding/json"
	"testing"
)

func TestPromptContentBlocksMixed(t *testing.T) {
	blocks := PromptContentBlocks("see ![](shot.png) please")
	if len(blocks) != 2 {
		t.Fatalf("len=%d %#v", len(blocks), blocks)
	}
	var text map[string]string
	if err := json.Unmarshal(blocks[0], &text); err != nil {
		t.Fatal(err)
	}
	if text["type"] != "text" || text["text"] != "see please" {
		t.Fatalf("%#v", text)
	}
	var img map[string]string
	if err := json.Unmarshal(blocks[1], &img); err != nil {
		t.Fatal(err)
	}
	if img["type"] != "image" || img["path"] != "shot.png" {
		t.Fatalf("%#v", img)
	}
}

func TestPromptContentBlocksImageOnly(t *testing.T) {
	blocks := PromptContentBlocks("![](a.png)")
	if len(blocks) != 1 {
		t.Fatalf("%d", len(blocks))
	}
	var img map[string]string
	_ = json.Unmarshal(blocks[0], &img)
	if img["path"] != "a.png" {
		t.Fatalf("%#v", img)
	}
}
