package platform

// BlankPreviewDocument is the sole document accepted by the non-Windows
// isolated helper. Application code may evaluate its fixed encoder script;
// markup, URLs, Host RPC and remote subresources are never supplied to it.
const BlankPreviewDocument = `<!doctype html><html><head><meta http-equiv="Content-Security-Policy" content="default-src 'none'; script-src 'none'; connect-src 'none'; img-src 'none'; media-src 'none'; frame-src 'none'; worker-src 'none'; object-src 'none'; form-action 'none'; base-uri 'none'"><title>Private encoder</title></head><body></body></html>`

func BlankPreviewOptions(o *WindowOptions) bool {
	return o != nil && o.IsolatedPreview && !o.BrowserProfile && !o.Surface && o.Native == 0 &&
		o.LocalPDFURL == "" && o.LocalPDFViewerURL == "" && o.LocalHTMLURL == "" && o.LocalMediaURL == ""
}
func BlankPreviewNavigation(target string) bool { return target == "about:blank" }
