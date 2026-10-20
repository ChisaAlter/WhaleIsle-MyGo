package chat

import "github.com/egoist/mygo/ui"

// Attachment kind glyphs: 16-unit outlines on currentColor, one stroke width,
// so the 28 DIP seat stays legible at the transcript's 13 DIP text scale.
var (
	attachmentIconFile = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linejoin="round">
    <path d="M9.2 1.8H4.4c-.6 0-1 .4-1 1v10.4c0 .6.4 1 1 1h7.2c.6 0 1-.4 1-1V5.2L9.2 1.8Z"/>
    <path d="M9.2 1.8v3.4h3.4"/>
  </svg>`))
	attachmentIconText = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linejoin="round">
    <path d="M9.2 1.8H4.4c-.6 0-1 .4-1 1v10.4c0 .6.4 1 1 1h7.2c.6 0 1-.4 1-1V5.2L9.2 1.8Z"/>
    <path d="M9.2 1.8v3.4h3.4"/>
    <path d="M5.6 8.6h4.8M5.6 11h3.4" stroke-linecap="round"/>
  </svg>`))
	attachmentIconDocument = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linejoin="round">
    <path d="M9.2 1.8H4.4c-.6 0-1 .4-1 1v10.4c0 .6.4 1 1 1h7.2c.6 0 1-.4 1-1V5.2L9.2 1.8Z"/>
    <path d="M9.2 1.8v3.4h3.4"/>
    <path d="M5.8 7.4h4.4M5.8 9.6h4.4M5.8 11.8h2.6" stroke-linecap="round"/>
  </svg>`))
	attachmentIconSheet = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linejoin="round">
    <rect x="2.4" y="2.6" width="11.2" height="10.8" rx="1.2"/>
    <path d="M2.4 6.2h11.2M2.4 9.6h11.2M6.4 2.6v10.8M9.8 2.6v10.8"/>
  </svg>`))
	attachmentIconArchive = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linejoin="round">
    <rect x="2.6" y="3" width="10.8" height="10" rx="1.2"/>
    <path d="M8 3v2.2M8 6.4v2.2M8 9.8v1.6" stroke-linecap="round"/>
    <rect x="6.8" y="6.2" width="2.4" height="2" rx=".4"/>
  </svg>`))
	attachmentIconCode = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round">
    <path d="M5.6 5.2 2.8 8l2.8 2.8M10.4 5.2 13.2 8l-2.8 2.8M9.2 3.4 6.8 12.6"/>
  </svg>`))
	attachmentIconImage = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linejoin="round">
    <rect x="2.4" y="3" width="11.2" height="10" rx="1.4"/>
    <circle cx="5.9" cy="6.2" r="1.1"/>
    <path d="m3.2 11.6 3.1-3.2 2.2 2.2 1.9-1.9 2.6 2.6"/>
  </svg>`))
	attachmentIconAudio = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linejoin="round">
    <path d="M9.4 2.6 4.6 6.2H2.8v3.6h1.8l4.8 3.6V2.6Z"/>
    <path d="M11.8 5.6a3.4 3.4 0 0 1 0 4.8M13.4 4a5.6 5.6 0 0 1 0 8" stroke-linecap="round"/>
  </svg>`))
	attachmentIconVideo = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linejoin="round">
    <rect x="1.8" y="4" width="8.6" height="8" rx="1.4"/>
    <path d="m10.4 8 3.8-2.2v4.4L10.4 8Z"/>
  </svg>`))
)
