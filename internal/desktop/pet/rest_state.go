package pet

import "time"

// restState is transient, owned by one page/window. The original renderer
// enters doze after 225 seconds and sleep after 240 seconds without contact.
// Neither state writes care stats; only an explicit successful wake does.
type restState struct {
	sleeping        bool
	dozing          bool
	message         string
	lastInteraction time.Time
}

func (r *restState) touch(now time.Time) {
	r.lastInteraction = now
	r.dozing = false
}

func (r *restState) advance(now time.Time, blocked bool) {
	if blocked || r.lastInteraction.IsZero() || now.Before(r.lastInteraction) {
		r.touch(now)
		return
	}
	if r.sleeping {
		return
	}
	idle := now.Sub(r.lastInteraction)
	r.dozing = idle > 225*time.Second && idle <= 240*time.Second
	if idle > 240*time.Second {
		r.sleep()
	}
}

func (r *restState) sleep()                         { r.sleeping = true; r.dozing = false; r.message = "" }
func (r *restState) wake(commit func() error) error { return r.wakeAt(time.Now(), commit) }
func (r *restState) wakeAt(now time.Time, commit func() error) error {
	if r.sleeping && commit != nil {
		if err := commit(); err != nil {
			return err
		}
	}
	r.sleeping = false
	r.message = ""
	r.touch(now)
	return nil
}

// restClock is UI-owned. Incrementing its generation also invalidates timer
// callbacks already enqueued by Window.Update before a stop/Host change.
type restClock struct {
	generation        uint64
	runtimeOwner      string
	runtimeGeneration int
	runtimeKnown      bool
	cancel            func()
}

func (c *restClock) stop() {
	c.generation++
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
}
