package components

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"
)

type bridgeState struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	PID     int    `json:"pid"`
	Ready   bool   `json:"ready"`
	URL     string `json:"url"`
}

var managementKeyPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

func readBridgeState(path string) (bridgeState, error) {
	var st bridgeState
	b, err := os.ReadFile(path)
	if err != nil {
		return st, err
	}
	if json.Unmarshal(b, &st) != nil {
		return st, fmt.Errorf("鲸桥状态文件无效")
	}
	return st, nil
}

// verifyBridgeReady uses only the authenticated loopback status endpoint. Errors
// never contain the management URL, token, response body, or credentials.
func verifyBridgeReady(ctx context.Context, st bridgeState, version string, pid int, client *http.Client) error {
	if st.ID != "whalebridge" || !st.Ready || st.PID <= 0 || st.PID != pid || st.Version != version {
		return fmt.Errorf("鲸桥进程 PID 或版本与启动记录不一致")
	}
	u, err := url.Parse(st.URL)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.Port() == "" || u.User != nil || !managementKeyPattern.MatchString(u.Query().Get("k")) {
		return fmt.Errorf("鲸桥本机管理地址无效")
	}
	key := u.Query().Get("k")
	endpoint := "http://" + u.Host + "/api/whalebridge/status"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("鲸桥状态请求无效")
	}
	req.AddCookie(&http.Cookie{Name: "magpie_web_" + u.Port(), Value: key})
	// Do not forward the authentication cookie through redirects or proxies.
	safe := *client
	safe.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	safe.Transport = &http.Transport{Proxy: nil}
	defer safe.Transport.(*http.Transport).CloseIdleConnections()
	resp, err := safe.Do(req)
	if err != nil {
		return fmt.Errorf("鲸桥本机状态接口不可用")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("鲸桥状态接口 HTTP %d", resp.StatusCode)
	}
	var status struct {
		PID     int    `json:"pid"`
		Version string `json:"version"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&status) != nil || status.PID != pid || status.Version != version {
		return fmt.Errorf("鲸桥状态接口 PID 或版本不一致")
	}
	return nil
}

func waitBridgeReady(ctx context.Context, path, version string, pid int, exited <-chan error, client *http.Client) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-exited:
			return fmt.Errorf("鲸桥启动进程已退出，请查看组件日志")
		default:
		}
		st, err := readBridgeState(path)
		if err == nil && st.Ready {
			if err := verifyBridgeReady(ctx, st, version, pid, client); err != nil {
				return err
			}
			select {
			case <-exited:
				return fmt.Errorf("鲸桥启动进程已退出，请查看组件日志")
			default:
				return nil
			}
		}
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("无法读取鲸桥就绪状态，请查看组件日志")
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("鲸桥启动超时或已取消，请查看组件日志")
		case <-exited:
			return fmt.Errorf("鲸桥启动进程已退出，请查看组件日志")
		case <-ticker.C:
		}
	}
}
