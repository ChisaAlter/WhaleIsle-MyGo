package voiceinput

import (
	"context"
	"math"
	"sync"
	"time"

	"whaleisle/internal/desktop/voicecapture"
)

type MicrophoneState struct {
	Selected                 voicecapture.Device
	Devices                  []voicecapture.Device
	Active, Loading, Missing bool
	Level                    float64
	Error                    string
}

// Microphone owns the client-local device seat and its settings-page preview.
// The application persists Selected in its existing local preferences; speech
// provider/language remain Host-owned. Opening/closing the page owns hardware.
type Microphone struct {
	mu         sync.Mutex
	state      MicrophoneState
	generation uint64
	cancel     context.CancelFunc
	done       chan struct{}
	preview    voicecapture.Preview
	save       func(voicecapture.Device) error
	changed    func()
}

func NewMicrophone(selected voicecapture.Device, save func(voicecapture.Device) error, changed func()) *Microphone {
	return &Microphone{state: MicrophoneState{Selected: selected}, save: save, changed: changed}
}

func (m *Microphone) notify() {
	if m.changed != nil {
		m.changed()
	}
}
func (m *Microphone) Snapshot() MicrophoneState {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.state
	s.Devices = append([]voicecapture.Device(nil), s.Devices...)
	if m.preview != nil {
		// Same -60..0 dB meter as the original eight-bar device preview.
		s.Level = math.Max(0, math.Min(1, (20*math.Log10(math.Max(m.preview.Level(), 0.00001))+60)/60))
		if err := m.preview.Err(); err != nil {
			s.Error = err.Error()
		}
	}
	return s
}

// SetActive is called for the actual visible voice settings page and window
// visibility. Retiring it cancels requests and releases input before returning.
func (m *Microphone) SetActive(active bool) {
	m.mu.Lock()
	if m.state.Active == active {
		m.mu.Unlock()
		return
	}
	m.state.Active = active
	m.generation++
	generation := m.generation
	oldCancel, oldPreview, oldDone := m.cancel, m.preview, m.done
	m.cancel, m.preview, m.done = nil, nil, nil
	m.state.Level, m.state.Loading = 0, active
	var ctx context.Context
	var done chan struct{}
	if active {
		ctx, m.cancel = context.WithCancel(context.Background())
		done = make(chan struct{})
		m.done = done
	}
	m.mu.Unlock()
	if oldCancel != nil {
		oldCancel()
	}
	if oldPreview != nil {
		oldPreview.Cancel()
	}
	if oldDone != nil {
		<-oldDone
	}
	if active {
		go func() { defer close(done); m.run(ctx, generation) }()
	}
	m.notify()
}

func (m *Microphone) Select(device voicecapture.Device) {
	m.mu.Lock()
	valid := device.ID == ""
	for _, choice := range m.state.Devices {
		if choice.ID == device.ID {
			device, valid = choice, true
			break
		}
	}
	if !valid || device.ID == m.state.Selected.ID {
		m.mu.Unlock()
		return
	}
	active := m.state.Active
	m.mu.Unlock()
	if device.ID == "" {
		device.Label = ""
	}
	if m.save != nil {
		if err := m.save(device); err != nil {
			m.mu.Lock()
			m.state.Error = err.Error()
			m.mu.Unlock()
			m.notify()
			return
		}
	}
	m.SetActive(false)
	m.mu.Lock()
	m.state.Selected, m.state.Error = device, ""
	m.mu.Unlock()
	if active {
		m.SetActive(true)
	} else {
		m.notify()
	}
}

func (m *Microphone) Refresh() {
	m.mu.Lock()
	active := m.state.Active
	m.mu.Unlock()
	if active {
		m.SetActive(false)
		m.SetActive(true)
	}
}

func (m *Microphone) currentLocked(ctx context.Context, generation uint64) bool {
	return ctx.Err() == nil && m.state.Active && m.generation == generation
}

func (m *Microphone) run(ctx context.Context, generation uint64) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		devices, err := voicecapture.InputDevices()
		m.mu.Lock()
		if !m.currentLocked(ctx, generation) {
			m.mu.Unlock()
			return
		}
		m.state.Loading = false
		if err != nil {
			m.state.Error = err.Error()
			m.mu.Unlock()
			m.notify()
			return
		}
		m.state.Devices = devices
		selected := m.state.Selected
		present := selected.ID == "" && len(devices) > 0
		for _, device := range devices {
			if device.ID == selected.ID {
				present = true
			}
		}
		m.state.Missing = !present
		old := m.preview
		if !present {
			m.preview = nil
			m.state.Error = ""
		}
		start := present && m.preview == nil
		m.mu.Unlock()
		if !present && old != nil {
			old.Cancel()
		}
		if start {
			preview, failure := voicecapture.StartPreview(ctx, selected)
			m.mu.Lock()
			if !m.currentLocked(ctx, generation) {
				m.mu.Unlock()
				if preview != nil {
					preview.Cancel()
				}
				return
			}
			if failure != nil {
				m.state.Error = failure.Error()
			} else {
				m.preview, m.state.Error = preview, ""
			}
			m.mu.Unlock()
			m.notify()
			// A failed acquisition requires the explicit Retry gesture. Device
			// enumeration alone must not repeatedly reopen a denied microphone.
			if failure != nil {
				return
			}
		}
		m.notify()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
