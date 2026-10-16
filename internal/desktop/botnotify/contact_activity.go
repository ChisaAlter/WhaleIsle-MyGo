package botnotify

// ContactActivity carries the already-read bot/activity batch to the directory.
// The notification tracker continues to consume only Rows; this adds no polling.
type ContactActivity struct {
	ReadInitialized          bool
	ActivitySeq, LastSeenSeq int
	Error                    string
	ID, SessionID            string
	ActivityAt               int64
	Attention                bool
	Preview                  string
}
