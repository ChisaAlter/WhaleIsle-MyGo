package files

// Runtime-only, owned by one FileTab. Preserve the original requested line
// until that tab has successfully loaded its text; do not clamp against an
// empty/loading preview. Failed or cancelled reads leave the request pending.
type textReadLine struct {
	line    int
	waiting bool
}

func (p *textReadLine) begin() { p.waiting = true }

func (p *textReadLine) deferRequest(line int) bool {
	if !p.waiting {
		p.line = 0
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
	p.line = 0
	return line
}

func (p *textReadLine) reset() { *p = textReadLine{} }
