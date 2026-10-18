package pet

import (
	_ "embed"
	"net/http"
	"time"
)

// careAnimation is published only after the corresponding persisted care or
// growth change succeeds. The renderer never awards care for playing a clip.
type careAnimation struct {
	ID    int64
	Kind  string
	Name  string
	Level int
}

func newCareAnimation(kind, name string, level int) careAnimation {
	return careAnimation{ID: time.Now().UnixMilli(), Kind: kind, Name: name, Level: level}
}

//go:embed assets/pet-care-animation.js
var careAnimationScript []byte

func serveCareAnimation(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path != "/pet-care-animation.js" {
		return false
	}
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	_, _ = w.Write(careAnimationScript)
	return true
}
