package botnotify

// ContactActivity carries the already-read bot/activity batch to the directory.
// The notification tracker continues to consume only Rows; this adds no polling.
type ContactActivity struct {
	ID, SessionID string
	ActivityAt    int64
	Attention     bool
	Preview       string
}
