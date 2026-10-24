package i18n

const (
	KeyToolImagePreview      = "chat.tool.image.preview"
	KeyToolImageClosePreview = "chat.tool.image.close_preview"
	KeyToolImageLabel        = "chat.tool.image.label"
	KeyToolImageOpen         = "chat.tool.image.open"
	KeyToolImageOpenNamed    = "chat.tool.image.open_named"
	KeyToolImageLoading      = "chat.tool.image.loading"
	KeyToolImageFailed       = "chat.tool.image.failed"
	KeyToolImageSize         = "chat.tool.image.size"
	KeyToolImageDownscaled   = "chat.tool.image.downscaled"
)

func init() {
	Register(KeyToolImagePreview, "原图预览", "Original image preview")
	Register(KeyToolImageClosePreview, "关闭原图预览", "Close original image preview")
	Register(KeyToolImageLabel, "图片", "Image")
	Register(KeyToolImageOpen, "查看原图", "View original")
	Register(KeyToolImageOpenNamed, "%s，点击查看原图", "%s, click to view original")
	Register(KeyToolImageLoading, "图片加载中…", "Loading image…")
	Register(KeyToolImageFailed, "图片加载失败，点击重试", "Image failed to load; click to retry")
	Register(KeyToolImageSize, "%sx%spx，%s 字节", "%sx%spx, %s bytes")
	Register(KeyToolImageDownscaled, "%sx%spx（由 %sx%spx 缩小），%s 字节", "%sx%spx (downscaled from %sx%spx), %s bytes")
}
