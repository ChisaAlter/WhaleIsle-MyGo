package pet

import (
	"testing"
	"time"
)

func TestCareAnimationEvent(t *testing.T) {
	event := newCareAnimation("token", "", 3)
	if event.Kind != "token" || event.Level != 3 || time.Now().UnixMilli()-event.ID > 1000 {
		t.Fatalf("invalid event: %+v", event)
	}
	p := &Companion{careAction: event}
	if got := p.careStatus().Action; got != event {
		t.Fatalf("action not published: %+v", got)
	}
}
