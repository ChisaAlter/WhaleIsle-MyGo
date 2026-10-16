// Package accountcontact implements the original account launcher's questionnaire.
package accountcontact

import (
	"fmt"
	"math"
	"net/url"
	"strings"
)

const DefaultURL = "https://trtgsjkv6r.feishu.cn/share/base/form/shrcnlCoGElW7MQznGy9r3YYXcg?hide_uid=1&hide_device_info=1&hide_harness_version=1"

type Context struct {
	UID, Version, DeviceInfo  string
	Width, Height, PixelRatio float64
}

// URL accepts display-safe context only; account credentials are never inputs.
func URL(destination, source string, context Context) (string, error) {
	if destination == "" {
		destination = DefaultURL
	}
	u, err := url.Parse(destination)
	if err != nil {
		return "", err
	}
	if u.Scheme != "https" || u.Host == "" || u.User != nil {
		return "", fmt.Errorf("invalid contact questionnaire URL")
	}
	q := u.Query()
	ratio := context.PixelRatio
	if math.IsNaN(ratio) || math.IsInf(ratio, 0) {
		ratio = 1
	}
	device := context.DeviceInfo
	width, height := math.Round(context.Width*ratio), math.Round(context.Height*ratio)
	if width > 0 && height > 0 {
		if device != "" {
			device += "; "
		}
		device += fmt.Sprintf("screen_resolution=%.0fx%.0f", width, height)
	}
	for name, value := range map[string]string{"uid": context.UID, "source": source, "harness_version": context.Version, "device_info": device} {
		q.Set("hide_"+name, "1")
		q.Del("prefill_" + name)
		if value != "" {
			q.Set("prefill_"+name, value)
		}
	}
	for _, name := range strings.Fields("os_version device_brand app_locale screen_resolution app_version device_model") {
		q.Del("hide_" + name)
		q.Del("prefill_" + name)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// Open is called only by an explicit contact click. Read failures are optional,
// as in the original desktop bridge; the sampled account/build stays unchanged.
func Open(context Context, readDevice func() (string, error), open func(string) error) error {
	if readDevice != nil {
		context.DeviceInfo, _ = readDevice()
	}
	destination, err := URL("", "", context)
	if err != nil {
		return err
	}
	return open(destination)
}
