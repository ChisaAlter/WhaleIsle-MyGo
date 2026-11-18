package i18n

// Copy for the isolated SVG image surface. The seat draws a placeholder while
// the renderer is starting and when it cannot produce an image, so the box says
// what a person is looking at instead of quoting the renderer.
const (
	KeySvgImageUnavailable = "svg.image_unavailable"
)

func init() {
	Register(KeySvgImageUnavailable, "图像无法显示。", "Image could not be displayed.")
}
