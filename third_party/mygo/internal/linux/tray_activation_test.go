//go:build linux && (amd64 || arm64)

package linux

import "testing"

type activationHandler struct{ clicks int }

func (h *activationHandler) Clicked()     { h.clicks++ }
func (*activationHandler) RightClicked()  {}
func (*activationHandler) DoubleClicked() {}

func TestTrayActivationRoutesOnlyLiveIndicator(t *testing.T) {
	previous := theBackend
	defer func() { theBackend = previous }()
	h := &activationHandler{}
	theBackend = &Backend{trays: map[int]*tray{7: {id: 7, ind: 42, h: h}}}
	dispatchTrayActivation(42, 7)
	dispatchTrayActivation(99, 7)
	dispatchTrayActivation(42, 8)
	delete(theBackend.trays, 7)
	dispatchTrayActivation(42, 7)
	if h.clicks != 1 {
		t.Fatalf("live indicator should activate exactly once, got %d", h.clicks)
	}
}

func TestTrayActivationConnectsOnlySupportedSignal(t *testing.T) {
	oldType, oldLookup, oldConnect, oldCallback := trayIndicatorGetType, traySignalLookup, gSignalConnectData, cbTrayActivate
	defer func() {
		trayIndicatorGetType, traySignalLookup, gSignalConnectData, cbTrayActivate = oldType, oldLookup, oldConnect, oldCallback
	}()
	trayIndicatorGetType = func() uintptr { return 12 }
	cbTrayActivate = 21
	var signalID uint32
	traySignalLookup = func(name *byte, typ uintptr) uint32 {
		if typ != 12 || *name != 'a' {
			t.Fatal("wrong signal lookup")
		}
		return signalID
	}
	connections := 0
	gSignalConnectData = func(instance ptr, signal *byte, handler, data, destroy ptr, flags int32) uint64 {
		connections++
		if instance != 42 || handler != 21 || data != 7 {
			t.Fatal("wrong activation connection")
		}
		return 1
	}
	tr := &tray{id: 7, ind: 42}
	tr.connectActivation()
	if connections != 0 {
		t.Fatal("legacy indicator must keep its menu-only behavior")
	}
	signalID = 1
	tr.connectActivation()
	if connections != 1 {
		t.Fatal("supported indicator did not connect activation")
	}
}
