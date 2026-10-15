package composer

// modelPickerChoice keeps toolkit input across input/paint passes, while
// committed values and session changes always come from the Store snapshot.
type modelPickerChoice struct {
	SessionID string
	Committed string
	Value     string
}

func (p *modelPickerChoice) sync(sessionID, committed string) {
	if p.SessionID != sessionID || p.Committed != committed {
		p.SessionID, p.Committed, p.Value = sessionID, committed, committed
	}
}

func (p *modelPickerChoice) take() string {
	chosen := p.Value
	p.Value = p.Committed
	return chosen
}
