package accountlogin

import (
	"errors"
	"net"
	"net/url"
	"strconv"
	"sync/atomic"
	"time"

	"whaleisle/internal/dsh/api"
	"whaleisle/internal/ui/i18n"
)

var clientVersion atomic.Value

func init() {
	i18n.Register("account-client.callback-origin", "账户登录需要当前 Host 的本机 HTTP 地址。", "Account login requires the current loopback HTTP Host origin.")
	i18n.Register("account-client.callback-port", "账户登录需要带有效端口的本机 Host 地址。", "Account login requires an explicit loopback Host port.")
	i18n.Register("account-client.information-unavailable", "账户信息链接暂不可用。", "Account information link is unavailable.")
	i18n.Register("account-client.signin-failed", "登录未完成，请重试。", "Sign-in failed. Try again.")
	i18n.Register("account-client.signin-expired", "登录链接已过期，请重新登录。", "The sign-in link expired. Start sign-in again.")
	i18n.Register("account-client.signin-no-response", "授权服务没有响应，请重试。", "The authorization service did not respond. Try again.")
}

func FailureMessage(phase, code string) string {
	if phase == "expired" {
		return i18n.T("account-client.signin-expired")
	}
	if phase != "failed" {
		return ""
	}
	if code == "no-response" {
		return i18n.T("account-client.signin-no-response")
	}
	return i18n.T("account-client.signin-failed")
}

// AuthorizeURL preserves the Host's attempt parameters while carrying the
// resolved application palette to the authorization page.
func AuthorizeURL(raw, scheme string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", errors.New(i18n.T("account-client.information-unavailable"))
	}
	q := u.Query()
	q.Set("theme", scheme)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// SetClientVersion supplies the desktop build, shared by every account caller.
func SetClientVersion(version string) { clientVersion.Store(version) }

func ClientMetadata() api.AccountClientMetadata {
	version, _ := clientVersion.Load().(string)
	if version == "" {
		version = "dev"
	}
	locale := "zh-CN"
	if i18n.Locale() == i18n.LocaleEN {
		locale = "en-US"
	}
	_, offset := time.Now().Zone()
	return api.AccountClientMetadata{Version: version, Locale: locale, TimezoneOffsetSeconds: float64(offset)}
}

// The Host owns /oauth/callback on its webServer. No desktop scheme or new
// listener is involved: the browser returns directly to that same Host origin.
func CallbackOrigin(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return "", errors.New(i18n.T("account-client.callback-origin"))
	}
	ip := net.ParseIP(u.Hostname())
	port, err := strconv.Atoi(u.Port())
	if (u.Hostname() != "localhost" && (ip == nil || !ip.IsLoopback())) || err != nil || port < 1 || port > 65535 {
		return "", errors.New(i18n.T("account-client.callback-port"))
	}
	return u.Scheme + "://" + u.Host, nil
}
