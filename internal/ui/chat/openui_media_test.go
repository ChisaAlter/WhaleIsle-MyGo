package chat

import (
	"testing"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/webview"
)

type inspectorMediaHostProbe struct {
	hidden, closed, presentations int
	position                      float64
}

func (p *inspectorMediaHostProbe) Alive() bool          { return p.closed == 0 }
func (p *inspectorMediaHostProbe) State() webview.State { return webview.State{} }
func (p *inspectorMediaHostProbe) SetVisible(visible bool) {
	if !visible {
		p.hidden++
	}
}
func (p *inspectorMediaHostProbe) PresentMedia(_, _ mygo.Rectangle, _ []mygo.Rectangle) (bool, error) {
	p.presentations++
	return true, nil
}
func (p *inspectorMediaHostProbe) Close() error {
	p.closed++
	p.position = 0
	return nil
}

func TestInspectorPickerHidesMediaWithoutReplacingPlaybackHost(t *testing.T) {
	host := &inspectorMediaHostProbe{position: 73.25}
	seat := &openUIMediaSeat{host: host}
	media := &openUIMedia{seats: map[string]*openUIMediaSeat{"video": seat}}
	bounds := mygo.Rectangle{X: 100, Y: 20, Width: 401, Height: 251}
	for range 3 {
		media.masked = true
		state, err := media.presentHost(seat, bounds, bounds)
		if err != nil || state != "hidden" || seat.host != host || host.closed != 0 || host.position != 73.25 {
			t.Fatalf("temporary mask lost player state: %q %v %+v", state, err, host)
		}
		media.masked = false
		state, err = media.presentHost(seat, bounds, bounds)
		if err != nil || state != "native-view-created" || seat.host != host || host.position != 73.25 {
			t.Fatalf("cancel did not restore same player: %q %v %+v", state, err, host)
		}
	}
	if host.hidden != 3 || host.presentations != 3 {
		t.Fatal(host)
	}
	media.suspend()
	if seat.host != nil || host.closed != 1 {
		t.Fatal("actual offscreen teardown did not release player")
	}
}
