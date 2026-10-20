package markdown

import "github.com/egoist/mygo/ui"

// Code-card toolbar artwork, matching ui-primitives CodeToolbar's 14px glyphs
// (copy / check / wrap / nowrap) at a 16-unit viewBox.
var (
	codeIconCopy = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke-width="1">
    <rect x="1.52075" y="4.07373" width="10.3932" height="10.3932" rx="2" stroke="currentColor" />
    <path d="M11.9792 1.53296C13.36 1.53296 14.4792 2.65225 14.4792 4.03296V9.42847C14.4792 10.3756 13.9521 11.1987 13.1755 11.6228V10.3298C13.3652 10.0787 13.4792 9.7674 13.4792 9.42847V4.03296C13.4792 3.20453 12.8077 2.53296 11.9792 2.53296H6.58374C6.27966 2.53301 5.99684 2.6235 5.7605 2.77905H4.42358C4.85652 2.03463 5.66056 1.53304 6.58374 1.53296H11.9792Z" fill="currentColor" />
  </svg>`))
	codeIconCheck = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke-width="1">
    <path d="M2.25 8.5L5.49732 11.7473C5.90519 12.1552 6.57263 12.1344 6.95426 11.7018L13.75 4" stroke="currentColor" />
  </svg>`))
	codeIconWrap = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke-width="1">
    <path d="M2 4.5H14" stroke="currentColor" />
    <path d="M2 8.5H9.5C10.8807 8.5 12 9.61929 12 11C12 12.3807 10.8807 13.5 9.5 13.5H8" stroke="currentColor" />
    <path d="M9.5 11.75L8.25 13.5L9.5 15" stroke="currentColor" />
    <path d="M2 12.5H5" stroke="currentColor" />
  </svg>`))
	codeIconNoWrap = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke-width="1">
    <path d="M1.5 4H14.5" stroke="currentColor" />
    <path d="M1.5 8H10" stroke="currentColor" />
    <path d="M12.5 8H14.5" stroke="currentColor" />
    <path d="M1.5 12H6" stroke="currentColor" />
    <path d="M9 12H14.5" stroke="currentColor" />
  </svg>`))
)
