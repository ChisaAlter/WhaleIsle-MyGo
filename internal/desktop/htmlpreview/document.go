package htmlpreview

import "html"

// DocumentCSP deliberately forbids all script execution. The document preview
// does not run capture's runtime isolation Probe (which also checks WebRTC), so
// resource filtering alone must not be treated as permission to execute HTML.
const DocumentCSP = "default-src 'none'; script-src 'none'; style-src 'unsafe-inline'; img-src data:; font-src data:; frame-src 'self' about:; connect-src 'none'; worker-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'"
const documentChildCSP = "default-src 'none'; script-src 'none'; style-src 'unsafe-inline'; img-src data:; font-src data:; frame-src 'none'; connect-src 'none'; worker-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'"

// DocumentShell displays only static markup from one authorized UTF-8 snapshot.
// The iframe is parser-created, with no bootstrap script and no sandbox grants.
// Attribute encoding prevents supplied markup from escaping its opaque srcdoc.
func DocumentShell(source string) string {
	child := `<meta charset="utf-8"><meta http-equiv="Content-Security-Policy" content="` + html.EscapeString(documentChildCSP) + `">` + source
	return `<!doctype html><meta charset="utf-8"><meta http-equiv="Content-Security-Policy" content="` + html.EscapeString(DocumentCSP) + `"><style>html,body{margin:0;width:100%;height:100%;overflow:hidden}iframe{display:block;border:0;width:100%;height:100%}</style><body><iframe sandbox="" referrerpolicy="no-referrer" srcdoc="` + html.EscapeString(child) + `"></iframe></body>`
}
