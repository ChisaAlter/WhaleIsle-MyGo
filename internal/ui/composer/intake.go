package composer

import (
	"fmt"
	"mime"
	"path/filepath"
	"strings"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// The composer's intake limits mirror the desktop product's documented image
// rules: 8 images per message, 10 MiB each and 20 MiB per message. The Host's
// imageLimits projection is not part of session.ViewState yet, so these are the
// documented values; when the projection lands, read it here instead.
const (
	composerMaxImages     = 8
	composerMaxImageBytes = int64(10 << 20)
	composerMaxTotalBytes = int64(20 << 20)
)

// composerIntake counts the staged images and their approximate byte total, so
// the drop overlay can say whether a new drop still fits.
func composerIntake(d *DraftState) (images int, bytes int64) {
	if d == nil {
		return 0, 0
	}
	for _, a := range d.Attachments {
		if !composerAttachmentIsImage(a) {
			continue
		}
		images++
		switch {
		case a.Upload.File.Bytes > 0:
			bytes += int64(a.Upload.File.Bytes)
		case a.ImageData != "":
			// Staged images arrive base64-encoded; the decoded size is what counts.
			bytes += int64(len(a.ImageData)) * 3 / 4
		}
	}
	return images, bytes
}

// composerAttachmentIsImage mirrors the rail's own image test.
func composerAttachmentIsImage(a session.ComposerAttachment) bool {
	if a.ImageData != "" || strings.HasPrefix(a.MediaType, "image/") {
		return true
	}
	return strings.HasPrefix(mime.TypeByExtension(strings.ToLower(filepath.Ext(a.Path))), "image/")
}

// composerDropWarning explains why one more image will be refused, or returns
// "" when the drop still fits. The overlay paints it in the rail's error ink.
func composerDropWarning(images int, bytes int64) string {
	switch {
	case images >= composerMaxImages:
		return fmt.Sprintf(i18n.T(i18n.KeyComposerDropFull), composerMaxImages)
	case bytes+composerMaxImageBytes > composerMaxTotalBytes:
		return fmt.Sprintf(i18n.T(i18n.KeyComposerDropTooLarge), composerFileSize(composerMaxTotalBytes))
	}
	return ""
}

// composerFileSize renders a byte budget the way the intake copy reads (10 MB).
func composerFileSize(n int64) string {
	const mb = int64(1 << 20)
	if n >= mb {
		return fmt.Sprintf("%.0f MB", float64(n)/float64(mb))
	}
	return fmt.Sprintf("%.0f KB", float64(n)/1024)
}
