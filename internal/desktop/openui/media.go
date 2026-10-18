package openui

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

// MediaRect is measured in the owned viewer iframe's CSS viewport pixels.
// It is never a native-window coordinate or handle. The trusted chat owner
// maps it into its own currently painted seat and intersects its native clip.
type MediaRect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type MediaViewport struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// MediaRequest deliberately carries no URL, headers, profile, native command,
// parent handle or cookies. Only the fixed YouTube player accepts this video ID.
type MediaRequest struct {
	Revision    uint64        `json:"revision"`
	Operation   string        `json:"operation"`
	ComponentID string        `json:"componentId"`
	VideoID     string        `json:"videoId,omitempty"`
	Title       string        `json:"title,omitempty"`
	Rect        MediaRect     `json:"rect"`
	Clip        MediaRect     `json:"clip"`
	Viewport    MediaViewport `json:"viewport"`
	Visible     bool          `json:"visible"`
}

func ValidateMediaRequest(e Event) error {
	m := e.Media
	if e.Type != "media" || m == nil || e.Sequence == 0 || e.Sequence > 9007199254740991 ||
		e.RequestID != "media-"+strconv.FormatUint(e.Sequence, 10) || m.Revision == 0 || m.Revision > 9007199254740991 {
		return errors.New("invalid OpenUI media identity")
	}
	component := strings.TrimPrefix(m.ComponentID, "youtube-")
	if component == m.ComponentID || len(m.ComponentID) > 80 || component == "" || component[0] == '0' || strings.Trim(component, "0123456789") != "" {
		return errors.New("invalid OpenUI media component")
	}
	if m.Operation == "remove" {
		return nil
	}
	if m.Operation != "upsert" || len(m.VideoID) != 11 || strings.Trim(m.VideoID, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-") != "" ||
		len(m.Title) > 512 || !utf8.ValidString(m.Title) || strings.ContainsRune(m.Title, '\x00') {
		return errors.New("invalid OpenUI YouTube parameters")
	}
	finite := func(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
	validRect := func(r MediaRect) bool {
		return finite(r.X) && finite(r.Y) && finite(r.Width) && finite(r.Height) && math.Abs(r.X) <= 1e6 && math.Abs(r.Y) <= 1e6 && r.Width >= 0 && r.Width <= 4096 && r.Height >= 0 && r.Height <= 4096
	}
	v := m.Viewport
	if !finite(v.Width) || !finite(v.Height) || v.Width < 1 || v.Width > 4096 || v.Height < 1 || v.Height > 4096 || !validRect(m.Rect) || !validRect(m.Clip) {
		return errors.New("invalid OpenUI media geometry")
	}
	c, r := m.Clip, m.Rect
	if c.X < 0 || c.Y < 0 || c.X+c.Width > v.Width+0.5 || c.Y+c.Height > v.Height+0.5 {
		return errors.New("OpenUI media clip escapes its viewer viewport")
	}
	if c.Width > 0 && c.Height > 0 && (c.X < r.X-0.5 || c.Y < r.Y-0.5 || c.X+c.Width > r.X+r.Width+0.5 || c.Y+c.Height > r.Y+r.Height+0.5) {
		return errors.New("OpenUI media clip escapes its component")
	}
	if m.Visible && (r.Width <= 0 || r.Height <= 0 || c.Width <= 0 || c.Height <= 0) {
		return errors.New("visible OpenUI media requires a nonempty clip")
	}
	return nil
}
