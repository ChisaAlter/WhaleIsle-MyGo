//go:build !windows

package update

import "net/http"

var updateProxy = http.ProxyFromEnvironment
