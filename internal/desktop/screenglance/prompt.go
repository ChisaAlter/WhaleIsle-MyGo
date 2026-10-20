package screenglance

import (
	"runtime"
	"whaleisle/internal/store/storelocale"
)

// ApprovalMessage describes the actual platform capture scope before any pixels
// are requested. The Screenshot portal, unlike Windows/macOS, owns selection.
func ApprovalMessage() string {
	if runtime.GOOS == "linux" {
		return storelocale.T(storelocale.KeyDesktopScreenglanceCaptureThroughTheSystemScreenshotDi)
	}
	return storelocale.T(storelocale.KeyDesktopScreenglanceCaptureAndSendOneScreenshotOfTheP)
}

func StorageDetail() string {
	switch runtime.GOOS {
	case "linux":
		return storelocale.T(storelocale.KeyDesktopScreenglanceTheImageMayContainPrivateInformatio)
	case "darwin":
		return storelocale.T(storelocale.KeyDesktopScreenglanceTheImageMayContainPrivateInformatio2)
	default:
		return storelocale.T(storelocale.KeyDesktopScreenglanceTheImageMayContainPrivateInformatio3)
	}
}

// FailureMessage avoids showing backend errors that could contain private paths.
func FailureMessage() string {
	switch runtime.GOOS {
	case "darwin":
		return storelocale.T(storelocale.KeyDesktopScreenglanceScreenshotUnavailableCheckThisAppS)
	case "linux":
		return storelocale.T(storelocale.KeyDesktopScreenglanceScreenshotCancelledDeniedOrUnavailab)
	default:
		return storelocale.T(storelocale.KeyDesktopScreenglanceScreenshotUnavailable)
	}
}
