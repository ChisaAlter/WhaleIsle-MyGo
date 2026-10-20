package storelocale

// HTML templates and desktop runtime errors that were written in both
// languages at once. Kept in their own block so the generated catalog files
// stay machine-owned.
const (
	KeyDesktopImagepreviewFitWidth         = "desktop.imagepreview.fit_width"
	KeyDesktopImagepreviewZoomOut          = "desktop.imagepreview.zoom_out"
	KeyDesktopImagepreviewZoomIn           = "desktop.imagepreview.zoom_in"
	KeyDesktopImagepreviewZoom             = "desktop.imagepreview.zoom"
	KeyDesktopImagepreviewImageZoom        = "desktop.imagepreview.image_zoom"
	KeyDesktopImagepreviewImageScrollArea  = "desktop.imagepreview.image_scroll_area"
	KeyDesktopImagepreviewIsolatedSnapshot = "desktop.imagepreview.isolated_snapshot"

	KeyDesktopPetSkinUnavailable       = "desktop.pet.skin_unavailable"
	KeyDesktopPetWindowCreateFailed    = "desktop.pet.window_create_failed"
	KeyDesktopPetAssetsMissing         = "desktop.pet.assets_missing"
	KeyDesktopPetOpenConversationTitle = "desktop.pet.open_conversation_title"
	KeyDesktopPetOpenChat              = "desktop.pet.open_chat"
)

func init() {
	register(KeyDesktopImagepreviewFitWidth, "适宽", "Fit width")
	register(KeyDesktopImagepreviewZoomOut, "缩小", "Zoom out")
	register(KeyDesktopImagepreviewZoomIn, "放大", "Zoom in")
	register(KeyDesktopImagepreviewZoom, "缩放", "Zoom")
	register(KeyDesktopImagepreviewImageZoom, "图片缩放", "Image zoom")
	register(KeyDesktopImagepreviewImageScrollArea, "图片滚动区域", "Image scroll area")
	register(KeyDesktopImagepreviewIsolatedSnapshot, "隔离图片快照。AVIF 支持取决于已安装的 WebView 运行时；图片损坏说明该编码未被解码。", "Isolated image snapshot. AVIF support depends on the installed WebView runtime; a broken image means that codec was not decoded.")

	register(KeyDesktopPetSkinUnavailable, "皮肤不可用", "Selected skin is unavailable")
	register(KeyDesktopPetWindowCreateFailed, "桌宠窗口创建失败", "Companion window could not open")
	register(KeyDesktopPetAssetsMissing, "缺少桌宠资源", "THA4 assets unavailable; stage pet-render-assets beside the executable")
	register(KeyDesktopPetOpenConversationTitle, "回到此会话", "Open this conversation")
	register(KeyDesktopPetOpenChat, "回到会话", "Open chat")
}
