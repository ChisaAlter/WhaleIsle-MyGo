package chrome

import (
	"os"
	"testing"

	"whaleisle/internal/desktop/media"
)

// Chrome's paint tests read pixels in the frame that requests a wallpaper:
// drive the media service synchronously for them.
func TestMain(m *testing.M) {
	s := media.New(media.Config{Workers: -1})
	wallpaperMedia = func() *media.Service { return s }
	wallpaperSettle = func(s *media.Service) { s.RunPending() }
	os.Exit(m.Run())
}
