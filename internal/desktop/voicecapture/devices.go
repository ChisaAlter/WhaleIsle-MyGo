package voicecapture

// Device is a client-local microphone preference, independent of the Host's
// speech provider/language selection. Empty ID follows the system input.
type Device struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Preview owns only a live input-level meter. It has no Stop/export/transcribe
// operation and retains no audio after each measured buffer.
type Preview interface {
	Level() float64
	Err() error
	Cancel()
}
