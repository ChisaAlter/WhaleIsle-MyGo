package browserpreview

import "errors"

// The original Browser external action uses the address-bar draft, even before
// an embedded page is created. Reuse the same HTTP(S), no-userinfo normalization
// as navigation; the desktop opener independently validates again before Shell.
func openBrowserAddress(draft string, open func(string) error) error {
	raw := NormalizeURL(draft)
	if raw == "" {
		return errors.New("请输入有效且不含账号密码的 HTTP(S) 地址")
	}
	if open == nil {
		return errors.New("系统浏览器打开功能不可用")
	}
	// URL only: no Browser profile, Cookie, auth header, file capability or
	// workspace state is handed off. This runs only after an explicit UI click.
	return open(raw)
}
