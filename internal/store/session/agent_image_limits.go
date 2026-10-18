package session

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
)

func cloneChildImageLimits(p *ComposerImageLimits) *ComposerImageLimits {
	if p == nil {
		return nil
	}
	v := *p
	v.MediaTypes = slices.Clone(p.MediaTypes)
	return &v
}

// This state belongs to the addressed child view, never Store.imageLimits (parent).
func seedChildImageLimits(v *AgentNavigationView, values json.RawMessage, seq *float64) {
	if !v.Address.validChild() {
		return
	}
	var block map[string]json.RawMessage
	if json.Unmarshal(values, &block) != nil {
		return
	}
	n := float64(0)
	if seq != nil {
		n = *seq
	}
	if n < v.imageLimitsSeq {
		return
	}
	// Reuse exactly the validated Host shape without borrowing parent scope.
	decoder := Store{selectedID: v.SessionID, imageLimits: composerImageLimitState{sessionID: v.SessionID, value: cloneChildImageLimits(v.ImageLimits), seq: v.imageLimitsSeq}}
	decoder.applyImageLimitsLocked(block["imageLimits"], n)
	v.ImageLimits = cloneChildImageLimits(decoder.imageLimits.value)
	v.imageLimitsSeq = decoder.imageLimits.seq
}
func validateChildImages(paths []string, limits *ComposerImageLimits) error {
	if limits != nil && int64(len(paths)) > limits.MaxImagesPerMessage {
		return fmt.Errorf("child image limit: maximum %d images", limits.MaxImagesPerMessage)
	}
	var total int64
	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		info, err := f.Stat()
		if err != nil {
			f.Close()
			return err
		}
		header := make([]byte, 512)
		n, _ := f.Read(header)
		f.Close()
		media := http.DetectContentType(header[:n])
		if !slices.Contains([]string{"image/png", "image/jpeg", "image/gif", "image/webp"}, media) {
			return fmt.Errorf("child continuation accepts images only: %s", info.Name())
		}
		total += info.Size()
		if limits != nil && (!slices.Contains(limits.MediaTypes, media) || info.Size() > limits.MaxImageBytes || total > limits.MaxMessageImageBytes) {
			return fmt.Errorf("image exceeds this child's supported type or size limits: %s", info.Name())
		}
	}
	return nil
}
