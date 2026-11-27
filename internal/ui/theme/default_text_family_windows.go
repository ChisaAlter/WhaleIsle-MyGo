//go:build windows

package theme

// DefaultTextFamily prioritizes the embedded Inter 400–700 Latin family on
// Windows so WeightMedium (500) does not collapse to Segoe UI Semibold (600),
// while preserving Segoe UI and CJK fallbacks (Microsoft YaHei) in order.
const DefaultTextFamily = `"Inter", -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', 'Helvetica Neue', Helvetica, Arial, sans-serif`
