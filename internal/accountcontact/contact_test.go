package accountcontact

import (
	"errors"
	"net/url"
	"testing"
)

func TestContactClickUsesSampledContextAndOptionalDevice(t *testing.T) {
	sampled := Context{UID: "uid-click", Version: "native-version", Width: 1920, Height: 1080, PixelRatio: 2}
	calls := 0
	err := Open(sampled, func() (string, error) { sampled.UID = "later-account"; return "", errors.New("unavailable") }, func(destination string) error {
		calls++
		u, err := url.Parse(destination)
		if err != nil {
			return err
		}
		q := u.Query()
		if u.Host != "trtgsjkv6r.feishu.cn" || q.Get("prefill_uid") != "uid-click" || q.Get("prefill_harness_version") != "native-version" || q.Get("prefill_device_info") != "screen_resolution=3840x2160" || q.Get("hide_source") != "1" || q.Has("prefill_source") {
			t.Fatalf("unexpected questionnaire context: %s", destination)
		}
		return nil
	})
	if err != nil || calls != 1 {
		t.Fatalf("open=%d err=%v", calls, err)
	}
}
