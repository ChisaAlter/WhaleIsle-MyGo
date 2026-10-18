package session

import (
	"encoding/json"
	"testing"
)

func TestComposerImageLimitsProjectionScope(t *testing.T) {
	s := New()
	s.selectedID = "A"
	s.followGeneration = 4
	s.applyImageLimitValuesLocked(json.RawMessage(`{"imageLimits":{"mediaTypes":["image/png"],"maxImagesPerMessage":2,"maxImageBytes":30,"maxMessageImageBytes":50}}`), 3)
	p := s.ComposerImageLimits("A")
	if p == nil || p.MaxImageBytes != 30 {
		t.Fatal(p)
	}
	p.MediaTypes[0] = "changed"
	if s.ComposerImageLimits("A").MediaTypes[0] != "image/png" {
		t.Fatal("mutable projection")
	}
	s.applyImageLimitControlLocked(json.RawMessage(`{"type":"projection","sessionId":"B","key":"imageLimits","seq":5,"value":null}`))
	if s.ComposerImageLimits("A") == nil {
		t.Fatal("cross-session control")
	}
	s.followGeneration++
	if s.ComposerImageLimits("A") != nil {
		t.Fatal("stale Host/follow limits")
	}
}
