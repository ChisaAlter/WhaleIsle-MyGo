package update

import "net/http"

// Every updater request (catalog, blockmap and installer) uses one proxy
// policy. Explicit environment proxies remain supported; Windows additionally
// honors the current user's system/PAC configuration.
func updateTransport() http.RoundTripper {
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return http.DefaultTransport
	} // An explicit custom transport keeps its existing contract.
	t := base.Clone()
	t.Proxy = updateProxy
	return t
}
