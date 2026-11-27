//go:build !windows

package theme

// DefaultTextFamily mirrors ui-theme/src/styles/base.css. MyGo resolves the
// comma-separated families in order, including CJK fallbacks after system sans.
const DefaultTextFamily = "-apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', 'Helvetica Neue', Helvetica, Arial, sans-serif"
