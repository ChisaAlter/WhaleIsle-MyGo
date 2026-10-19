package files

// Runtime-only, owned by one FileTab. Preserve the original requested line
// until that tab has successfully loaded its text; do not clamp against an
// empty/loading preview. Failed or cancelled reads leave the request pending.
type textReadLine struct {
	line    int
	waiting bool
	resume  bool // A cancelled page, not a failed page, may resume on tab return.
	version string
}

func (p *textReadLine) begin() { p.waiting = true }

func (p *textReadLine) deferRequest(line int) bool {
	if !p.waiting {
		p.reset()
		return false
	}
	p.line = max(1, line)
	return true
}

func (p *textReadLine) complete(success bool) int {
	if !success {
		return 0
	}
	p.waiting = false
	line := p.line
	p.reset()
	return line
}

// Keep one navigation attached to its loaded resource version. A first page
// establishes the version; a reload must not land an old target on new bytes.
func (p *textReadLine) acceptVersion(version string) bool {
	if p.line == 0 {
		return true
	}
	if p.version != "" && p.version != version {
		p.reset()
		return false
	}
	p.version = version
	return true
}

func (p *textReadLine) reset() { *p = textReadLine{} }
