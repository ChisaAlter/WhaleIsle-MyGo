package settings

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/media"
)

// RetryAccountAvatar is the explicit refresh of the account avatar shared by
// the settings page and the sidebar.
func RetryAccountAvatar(raw string, invalidate func()) {
	media.Default().RetryAvatar(raw)
	if invalidate != nil {
		invalidate()
	}
}

// accountAvatar reads the shared avatar handle; decoding and downloads run on
// the media service.
func accountAvatar(_ *ui.Context, raw string) *ui.Bitmap {
	return media.Default().Avatar(raw).Bitmap()
}
