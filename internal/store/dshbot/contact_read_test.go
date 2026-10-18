package dshbot

import "testing"

func TestContactUnreadHostCursor(t *testing.T) {
	for _, tt := range []struct {
		a    ContactActivity
		want bool
	}{
		{ContactActivity{ReadInitialized: true, ActivitySeq: 9, LastSeenSeq: 8}, true},
		{ContactActivity{ReadInitialized: true, ActivitySeq: 9, LastSeenSeq: 9}, false},
		{ContactActivity{ActivitySeq: 9}, false},
		{ContactActivity{ReadInitialized: true, ActivitySeq: 9, LastSeenSeq: 10}, false},
		{ContactActivity{ReadInitialized: true, ActivitySeq: 9, LastSeenSeq: 8, Active: true}, false},
	} {
		if tt.a.Unread() != tt.want {
			t.Fatalf("wrong unread %+v", tt)
		}
	}
}
