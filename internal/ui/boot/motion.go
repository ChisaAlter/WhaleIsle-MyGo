package boot

import "time"

// motionStep paces every boot animation to one shared 20 fps tick. The page
// is up for the whole Host start (10-20 s); driving it at the display's
// refresh rate repainted the full window 130-150 times a second for motion
// that moves a few DIP per second. 20 fps leaves room under the 30 fps
// startup budget for the log progress (5/s) and store work. All animations
// read the same quantized clock and ask for the same next tick, so their
// frames coalesce.
const motionStep = time.Second / 20

// motionNow is now rounded down to the current tick.
func motionNow(now time.Time) time.Time { return now.Truncate(motionStep) }

// motionWait is the delay to the next tick.
func motionWait(now time.Time) time.Duration {
	return max(time.Millisecond, motionStep-now.Sub(now.Truncate(motionStep)))
}

// loopPhase is the position, 0 to 1, of the current tick in a repeating
// period, read from the shared clock so every animation lands on one frame.
func loopPhase(now time.Time, period time.Duration) float32 {
	period = max(period, motionStep)
	return float32(motionNow(now).UnixNano()%int64(period)) / float32(period)
}
