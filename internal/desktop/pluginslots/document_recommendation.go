package pluginslots

// DocumentRecommendation is presentation-only state from the exact document
// page. It cannot request navigation, RPC, filesystem access or script execution.
type DocumentRecommendation struct {
	Session     string `json:"session"`
	Path        string `json:"path"`
	Spreadsheet bool   `json:"spreadsheet"`
	Office      bool   `json:"office"`
	Extension   bool   `json:"extension"`
}

func (g *Gateway) ConfigureDocumentViewer(session, path string) {
	g.modules.mu.Lock()
	g.modules.documentSession, g.modules.documentPath = session, path
	g.modules.mu.Unlock()
}

func (b *clientModulesBridge) acceptsDocument(message *DocumentRecommendation) bool {
	return message != nil && b.documentSession != "" && message.Session == b.documentSession && message.Path == b.documentPath
}
