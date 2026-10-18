//go:build linux && (amd64 || arm64)

package linux

import (
	"fmt"
	"strconv"
	"strings"
)

// RequestDesktopScreenshot must run on the GTK main thread. It shares MyGo's
// registered portal connection and signal dispatcher. The desktop owns the
// selection/permission UI; there is deliberately no direct X11 fallback.
func RequestDesktopScreenshot(done func(string, error)) (func(), error) {
	conn, err := bus()
	if err != nil {
		return nil, err
	}
	if portalVersion("org.freedesktop.portal.Screenshot") < 2 {
		return nil, fmt.Errorf("interactive Screenshot portal v2 is unavailable")
	}
	if !portalListening {
		gDBusConnectionSignalSubscribe(conn, cs(portalName), cs("org.freedesktop.portal.Request"), cs("Response"), nil, nil, 0, cbPortalSignal, 0, 0)
		portalListening = true
	}
	portalTokens++
	token := "mygoscreenshot" + strconv.Itoa(portalTokens)
	sender := strings.ReplaceAll(strings.TrimPrefix(goStr(gDBusConnectionGetUniqueName(conn)), ":"), ".", "_")
	path := portalPath + "/request/" + sender + "/" + token
	finished := false
	response := func(code uint32, results ptr) {
		finished = true
		if code == 1 {
			done("", fmt.Errorf("screenshot cancelled"))
			return
		}
		if code != 0 {
			done("", fmt.Errorf("screenshot permission denied or capture failed"))
			return
		}
		uri := vardictString(results, "uri")
		if uri == "" {
			done("", fmt.Errorf("screenshot portal returned no image"))
			return
		}
		done(uri, nil)
	}
	portalRequests[path] = response
	params := tuple(gVariantNewString(cs("")), vardict(
		vardictEntry("handle_token", gVariantNewString(cs(token))),
		vardictEntry("interactive", gVariantNewBoolean(true)),
		vardictEntry("modal", gVariantNewBoolean(true)),
	))
	res, err := portalCall(portalPath, "org.freedesktop.portal.Screenshot", "Screenshot", params, "(o)")
	if err != nil {
		delete(portalRequests, path)
		return nil, err
	}
	handle := gVariantGetChildValue(res, 0)
	actual := goStr(gVariantGetString(handle, nil))
	gVariantUnref(handle)
	gVariantUnref(res)
	if actual != path {
		delete(portalRequests, path)
		path = actual
		portalRequests[path] = response
	}
	return func() {
		if finished {
			return
		}
		finished = true
		delete(portalRequests, path)
		if res, err := portalCall(path, "org.freedesktop.portal.Request", "Close", 0, ""); err == nil {
			gVariantUnref(res)
		}
	}, nil
}
