//go:build windows

package d3d11

import (
	"math"
	"runtime"
	"testing"
	"time"
	"unsafe"

	"github.com/egoist/mygo/internal/gpu/gputest"
	"github.com/egoist/mygo/internal/scene"
)

var (
	procCreateSemaphoreW = kernel32.NewProc("CreateSemaphoreW")
	procReleaseSemaphore = kernel32.NewProc("ReleaseSemaphore")
)

func newTestSemaphore(t *testing.T, initial, maximum int32) uintptr {
	t.Helper()
	h, _, err := procCreateSemaphoreW.Call(0, uintptr(initial), uintptr(maximum), 0)
	if h == 0 {
		t.Fatalf("CreateSemaphoreW failed: %v", err)
	}
	t.Cleanup(func() { procCloseHandle.Call(h) })
	return h
}

func releaseTestSemaphore(t *testing.T, h uintptr, n int32) int32 {
	t.Helper()
	var prev int32
	ok, _, err := procReleaseSemaphore.Call(h, uintptr(n), uintptr(unsafe.Pointer(&prev)))
	if ok == 0 {
		t.Fatalf("ReleaseSemaphore(%d) failed: %v", n, err)
	}
	return prev
}

// peekTestSemaphoreCount inspects the exact current count of a Win32 semaphore
// non-destructively by draining and restoring it.
func peekTestSemaphoreCount(t *testing.T, h uintptr) int {
	t.Helper()
	n := 0
	for range 32 {
		ret, _, _ := procWaitForSingleObject.Call(h, 0)
		if ret != waitObject0 {
			break
		}
		n++
	}
	if n > 0 {
		releaseTestSemaphore(t, h, int32(n))
	}
	return n
}

func TestWaitableSwapChainModesAndResize(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	cases := []struct {
		name string
		make func(hwnd uintptr) (*Renderer, error)
	}{
		{"hwnd", New},
		{"composed", NewComposed},
		{"composed-rounded", func(hwnd uintptr) (*Renderer, error) { return NewComposedRounded(hwnd, 12) }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hwnd := hiddenWindow(t, 240, 160)
			r, err := tc.make(hwnd)
			if err != nil {
				t.Skip("no Direct3D 11:", err)
			}
			defer r.Release()

			// Before the first frame creates the swap chain, WaitHandle is 0 and
			// TryAcquireFrame never blocks the initial render.
			if r.WaitHandle() != 0 {
				t.Fatalf("expected 0 wait handle before first draw, got %#x", r.WaitHandle())
			}
			if !r.TryAcquireFrame() {
				t.Fatal("TryAcquireFrame must succeed before swap chain exists")
			}

			s1 := &scene.Scene{Width: 240, Height: 160, Clear: scene.Color{R: 20, G: 40, B: 60, A: 255}}
			if err := r.Render(s1); err != nil {
				t.Fatalf("first render failed: %v", err)
			}
			if r.swapChain2 == 0 {
				t.Skip("no IDXGISwapChain2 on this system")
			}
			if r.scFlags&dxgiSwapChainFlagWaitable == 0 || r.WaitHandle() == 0 {
				t.Fatalf("expected waitable swap chain handle, flags=%#x handle=%#x", r.scFlags, r.WaitHandle())
			}

			// MarkFrameReady must mark the next slot acquired without blocking.
			r.MarkFrameReady()
			if !r.TryAcquireFrame() {
				t.Fatal("TryAcquireFrame should return true after MarkFrameReady")
			}

			// Resizing must preserve scFlags across ResizeBuffers without DXGI errors.
			s2 := &scene.Scene{Width: 320, Height: 220, Clear: scene.Color{R: 10, G: 20, B: 30, A: 255}}
			if err := r.Render(s2); err != nil {
				t.Fatalf("resized render failed: %v", err)
			}
			if r.slotAcquired {
				t.Fatal("slotAcquired should be cleared after Present")
			}
			if r.WaitHandle() == 0 {
				t.Fatal("waitable handle lost after resize")
			}

			r.Release()
			if r.WaitHandle() != 0 {
				t.Fatalf("waitable handle not cleared after Release: %#x", r.WaitHandle())
			}
		})
	}
}

// TestWaitableSemaphoreNoDriftUnderForcedSequences deterministically simulates
// the exact failure sequences that previously caused permanent DXGI waitable
// semaphore drift (S+I >= 2):
//  1. Initial frame presented before TryAcquireFrame is called;
//  2. Forced/immediatePaint presents while a prior frame is still in flight;
//  3. Consecutive resize/deferredForce presents interleaved with DWM completions;
//  4. Event-loop MarkFrameReady wakeups draining excess/debt counts.
func TestWaitableSemaphoreNoDriftUnderForcedSequences(t *testing.T) {
	sem := newTestSemaphore(t, 1, 16)
	r := &Renderer{waitable: sem}

	// Simulate the pre/post Present bookkeeping around a mock DWM queue of
	// in-flight frames (where each completed frame releases +1 on sem).
	inFlight := 0
	simulatePresent := func(waitForPrior bool) {
		acquired := r.slotAcquired
		r.slotAcquired = false
		if !acquired {
			for r.unpacedPresents > 0 && r.waitFrameNonBlocking() {
				r.unpacedPresents--
			}
			if r.unpacedPresents == 0 && r.waitFrameNonBlocking() {
				acquired = true
			}
		}
		if acquired {
			r.drainExtraWaitable()
		}
		// Enqueue frame; if Present(1,0) had to block for an in-flight frame,
		// DWM completes that prior frame before Present returns.
		inFlight++
		if waitForPrior && inFlight > 1 {
			inFlight--
			releaseTestSemaphore(t, sem, 1)
		}
		now := time.Now()
		backToBack := !r.lastPresentAt.IsZero() && now.Sub(r.lastPresentAt) < 30*time.Millisecond
		r.lastPresentAt = now
		if !acquired {
			if !r.waitFrameNonBlocking() && backToBack {
				r.unpacedPresents = 1
			}
		}
	}
	completeOne := func() {
		if inFlight == 0 {
			return
		}
		inFlight--
		releaseTestSemaphore(t, sem, 1)
	}
	completeAll := func() {
		for inFlight > 0 {
			completeOne()
		}
	}
	assertConserved := func(step string) {
		t.Helper()
		s := peekTestSemaphoreCount(t, sem)
		held := 0
		if r.slotAcquired {
			held = 1
		}
		total := s + inFlight + held - r.unpacedPresents
		if total != 1 {
			t.Fatalf("%s: semaphore invariant violated: S=%d inFlight=%d held=%d debt=%d (net=%d, want 1)",
				step, s, inFlight, held, r.unpacedPresents, total)
		}
	}

	// 1. First frame presented without prior TryAcquireFrame (initial S=1).
	simulatePresent(false)
	assertConserved("first-frame")
	if r.TryAcquireFrame() {
		t.Fatal("first-frame: TryAcquireFrame must return false while frame 1 is in flight")
	}
	completeOne()
	assertConserved("first-frame-done")

	// 2. Immediate/resize present while idle (S=1), skipping TryAcquireFrame.
	simulatePresent(false)
	assertConserved("idle-immediate-paint")
	if r.TryAcquireFrame() {
		t.Fatal("idle-immediate-paint: TryAcquireFrame must return false while frame is in flight")
	}

	// 3. Forced present while frame is STILL in flight (S=0), sync=0 (non-blocking).
	// Now 2 frames are in flight and debt=1.
	simulatePresent(false)
	assertConserved("in-flight-resize-sync0")
	if r.unpacedPresents != 1 {
		t.Fatalf("expected unpacedPresents=1 after async unpaced present, got %d", r.unpacedPresents)
	}

	// Prior frame completes (S becomes 1, but second frame is STILL in flight!).
	// TryAcquireFrame MUST absorb the prior frame's signal to pay debt and
	// return false instead of letting the next Present(1,0) block!
	completeOne()
	if r.TryAcquireFrame() {
		t.Fatal("TryAcquireFrame must not succeed while the second in-flight frame has not completed")
	}
	assertConserved("debt-repaid-still-in-flight")

	// Second frame completes; MsgWait wakes and calls MarkFrameReady().
	completeOne()
	ret, _, _ := procWaitForSingleObject.Call(sem, 0)
	if ret != waitObject0 {
		t.Fatal("expected semaphore signaled after second frame completed")
	}
	r.MarkFrameReady()
	assertConserved("mark-frame-ready-after-debt")
	if !r.TryAcquireFrame() {
		t.Fatal("TryAcquireFrame must succeed after MarkFrameReady")
	}
	simulatePresent(false)
	assertConserved("paced-frame-after-recovery")

	// 4. Forced present with sync=1 (deferredForce / inMenu) while a frame is in flight:
	// Present blocks for the prior frame, so post-present reconciliation absorbs it at once.
	simulatePresent(true)
	assertConserved("deferred-force-sync1-reconciled")
	if r.unpacedPresents != 0 {
		t.Fatalf("expected immediate reconciliation after blocking sync=1 present, got debt=%d", r.unpacedPresents)
	}
	if r.TryAcquireFrame() {
		t.Fatal("TryAcquireFrame must still return false while the newly presented frame is in flight")
	}
	completeAll()
	assertConserved("idle-after-deferred-force")

	// 5. Simulate external semaphore drift (e.g. count inflated to 4) and verify
	// TryAcquireFrame / MarkFrameReady / present immediately drain it back to 1:1.
	releaseTestSemaphore(t, sem, 3)
	if !r.TryAcquireFrame() {
		t.Fatal("TryAcquireFrame failed with drifted semaphore")
	}
	if s := peekTestSemaphoreCount(t, sem); s != 0 {
		t.Fatalf("TryAcquireFrame did not drain excess drift: remaining S=%d", s)
	}
	simulatePresent(false)
	completeAll()
	if s := peekTestSemaphoreCount(t, sem); s != 1 {
		t.Fatalf("steady-state idle semaphore count = %d, want 1", s)
	}

	// 6. Simulate MarkFrameReady waking when debt=1 and only the first of two
	// in-flight frames finished: slotAcquired must stay false until the second finishes.
	if !r.TryAcquireFrame() {
		t.Fatal("expected paced acquire")
	}
	simulatePresent(false) // frame A
	simulatePresent(false) // forced frame B while A in flight -> debt=1
	completeOne()          // A completes -> S=1
	ret, _, _ = procWaitForSingleObject.Call(sem, 0)
	if ret != waitObject0 {
		t.Fatal("expected frame A signal")
	}
	r.MarkFrameReady()
	if r.slotAcquired {
		t.Fatal("MarkFrameReady must not mark slot ready when wakeup only repaid debt")
	}
	completeOne() // B completes -> S=1
	ret, _, _ = procWaitForSingleObject.Call(sem, 0)
	if ret != waitObject0 {
		t.Fatal("expected frame B signal")
	}
	r.MarkFrameReady()
	if !r.slotAcquired {
		t.Fatal("MarkFrameReady must mark slot ready once all debt is repaid")
	}
	simulatePresent(false)
	completeAll()
	if s := peekTestSemaphoreCount(t, sem); s != 1 {
		t.Fatalf("final steady-state semaphore count = %d, want 1", s)
	}

	// 7. Timeout fallback (>30ms after previous Present) with empty queue must
	// not re-arm unpacedPresents and wedge future frames in a 32ms loop.
	if !r.TryAcquireFrame() {
		t.Fatal("expected ready token before timeout recovery check")
	}
	r.slotAcquired = false // simulate a dropped/coalesced signal (S=0, inFlight=0)
	r.lastPresentAt = time.Now().Add(-40 * time.Millisecond)
	simulatePresent(false)
	if r.unpacedPresents != 0 {
		t.Fatalf("timeout recovery armed unpacedPresents=%d, want 0", r.unpacedPresents)
	}
	completeAll()
	if !r.TryAcquireFrame() {
		t.Fatal("next frame after timeout recovery failed to acquire slot")
	}
}

// TestWaitableRealSwapChainNoDrift exercises real D3D11 hwnd and DirectComposition
// swap chains across initial frame, repeated forced/immediate renders, resizes,
// and paced frames, verifying that the DXGI waitable semaphore never drifts >= 2
// and never starves subsequent paced frames.
func TestWaitableRealSwapChainNoDrift(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	for _, tc := range []struct {
		name string
		make func(hwnd uintptr) (*Renderer, error)
	}{
		{"hwnd", New},
		{"composed", NewComposed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hwnd := hiddenWindow(t, 240, 160)
			r, err := tc.make(hwnd)
			if err != nil {
				t.Skip("no Direct3D 11:", err)
			}
			defer r.Release()

			// 1. First frame + 8 forced back-to-back renders (simulating
			// immediatePaint / inMenu / deferredForce skipping TryAcquireFrame).
			base := &scene.Scene{Width: 240, Height: 160, Clear: scene.Color{R: 30, G: 60, B: 90, A: 255}}
			for i := range 9 {
				if err := r.Render(base); err != nil {
					t.Fatalf("forced render %d failed: %v", i, err)
				}
			}
			if r.swapChain2 == 0 || r.WaitHandle() == 0 {
				t.Skip("no waitable IDXGISwapChain2 on this system")
			}

			// 2. Repeated resizes (sync=0) interleaved with same-size renders (sync=1).
			for i := range 8 {
				s := &scene.Scene{Width: 240 + (i%4)*32, Height: 160 + (i%3)*24, Clear: scene.Color{R: 40, G: 50, B: 70, A: 255}}
				if err := r.Render(s); err != nil {
					t.Fatalf("resize render %d failed: %v", i, err)
				}
			}

			// 3. Wait for the swap chain to become ready (must not starve) and
			// verify TryAcquireFrame leaves zero excess count in the semaphore.
			deadline := time.Now().Add(2 * time.Second)
			for !r.TryAcquireFrame() {
				if time.Now().After(deadline) {
					t.Fatal("waitable swap chain starved after forced/resize sequence")
				}
				time.Sleep(time.Millisecond)
			}
			if extra, _, _ := procWaitForSingleObject.Call(r.WaitHandle(), 0); extra == waitObject0 {
				t.Fatal("DXGI waitable semaphore still had excess count after TryAcquireFrame")
			}

			// 4. Subsequent paced render cycle must complete cleanly and signal again.
			if err := r.Render(base); err != nil {
				t.Fatalf("paced render failed: %v", err)
			}
			deadline = time.Now().Add(2 * time.Second)
			for !r.TryAcquireFrame() {
				if time.Now().After(deadline) {
					t.Fatal("waitable swap chain did not signal after paced render")
				}
				time.Sleep(time.Millisecond)
			}
		})
	}
}

// TestComputeBlurTapsAccuracy verifies that precomputed bilinear-merged 1D
// Gaussian weights match discrete Gaussian convolution across representative
// radii/sigmas (including 100%/150%/200% DPI glass presets, odd/even radii,
// sigma=0 box fallback, and boundary clamping).
func TestComputeBlurTapsAccuracy(t *testing.T) {
	cases := []struct {
		name   string
		radius int
		sigma  float32
	}{
		{"small-even", 4, 1.3},
		{"small-odd", 5, 1.6},
		{"glass-20dip-150dpi", 12, 3.74},
		{"glass-20dip-200dpi", 15, 4.99},
		{"glass-40dip-150dpi", 23, 7.50},
		{"glass-40dip-200dpi", 30, 10.0},
		{"box-sigma-zero", 7, 0},
		{"max-inline", maxBlurTaps * 2, 42.0},
	}

	// Deterministic high-contrast test signal with sharp edges and gradients.
	const n = 48
	var sig [n]float64
	for i := range n {
		switch {
		case i < 8:
			sig[i] = 1.0
		case i%5 == 0:
			sig[i] = 0.85
		case i > 38:
			sig[i] = 0.25
		default:
			sig[i] = float64(i%7) / 6.0
		}
	}
	sampleLinear := func(x float64) float64 {
		// Mirror shader clamp: center coords clamped to [0, n-1].
		if x <= 0 {
			return sig[0]
		}
		if x >= float64(n-1) {
			return sig[n-1]
		}
		i0 := int(math.Floor(x))
		i1 := min(i0+1, n-1)
		f := x - float64(i0)
		return sig[i0]*(1-f) + sig[i1]*f
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			centerW, count, taps := computeBlurTaps(tc.radius, tc.sigma)
			wantCount := (tc.radius + 1) / 2
			if count != wantCount {
				t.Fatalf("tapCount = %d, want %d", count, wantCount)
			}
			totalW := float64(centerW)
			for k := range count {
				totalW += 2 * float64(taps[k][1])
			}
			if math.Abs(totalW-1.0) > 1e-5 {
				t.Fatalf("normalized weight sum = %v, want 1.0", totalW)
			}

			// Compare bilinear merged convolution against exact discrete convolution
			// at every pixel including boundaries (i = 0 .. n-1).
			var discW []float64
			discSum := float64(scene.BlurWeight(0, tc.sigma))
			discW = append(discW, discSum)
			for o := 1; o <= tc.radius; o++ {
				w := float64(scene.BlurWeight(o, tc.sigma))
				discW = append(discW, w)
				discSum += 2 * w
			}
			maxErr := 0.0
			for i := range n {
				want := sig[i] * discW[0]
				for o := 1; o <= tc.radius; o++ {
					l := max(i-o, 0)
					r := min(i+o, n-1)
					want += (sig[l] + sig[r]) * discW[o]
				}
				want /= discSum

				got := sig[i] * float64(centerW)
				for k := range count {
					off := float64(taps[k][0])
					w := float64(taps[k][1])
					got += (sampleLinear(float64(i)+off) + sampleLinear(float64(i)-off)) * w
				}
				maxErr = max(maxErr, math.Abs(got-want))
			}
			if maxErr > 1e-5 {
				t.Fatalf("max convolution error = %g, want <= 1e-5", maxErr)
			}
		})
	}

	// Oversized radius beyond maxBlurTaps*2 must return tapCount == 0 for shader fallback.
	if _, count, _ := computeBlurTaps(maxBlurTaps*2+1, 50); count != 0 {
		t.Fatalf("oversized radius returned tapCount=%d, want 0 (fallback)", count)
	}
}

// TestDrawHotPathZeroAllocs verifies that steady-state D3D11 rendering with
// fixed-arity COM call helpers and Renderer scratch storage produces zero heap
// allocations per frame.
func TestDrawHotPathZeroAllocs(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	s := gputest.Scene()
	r, err := New(hiddenWindow(t, s.Width, s.Height))
	if err != nil {
		t.Skip("no Direct3D 11:", err)
	}
	defer r.Release()

	// Warm up swap chain, textures, instance buffer, effect shaders, and backdrop cache.
	for range 3 {
		if err := r.draw(s); err != nil {
			t.Fatal(err)
		}
		if err := r.present(0); err != nil {
			t.Fatal(err)
		}
	}

	allocs := testing.AllocsPerRun(50, func() {
		if err := r.draw(s); err != nil {
			t.Fatal(err)
		}
		if err := r.present(0); err != nil {
			t.Fatal(err)
		}
	})
	if allocs > 0 {
		t.Fatalf("steady-state draw+present (bdCache hit) allocated %.2f objects/frame, want 0", allocs)
	}

	// Also verify zero allocations when backdrops are recomputed every frame
	// (readBackdrop + downPS + 2x blurPS).
	prevOff := backdropCacheOff
	backdropCacheOff = true
	defer func() { backdropCacheOff = prevOff }()
	allocsMiss := testing.AllocsPerRun(20, func() {
		if err := r.draw(s); err != nil {
			t.Fatal(err)
		}
		if err := r.present(0); err != nil {
			t.Fatal(err)
		}
	})
	if allocsMiss > 0 {
		t.Fatalf("steady-state draw+present (bdCache miss) allocated %.2f objects/frame, want 0", allocsMiss)
	}
}

func BenchmarkDrawHotPath(b *testing.B) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	s := gputest.Scene()
	hwnd, err := newHiddenWindow(s.Width, s.Height)
	if err != nil {
		b.Fatal(err)
	}
	defer destroyWindow(hwnd)
	r, err := New(hwnd)
	if err != nil {
		b.Skip("no Direct3D 11:", err)
	}
	defer r.Release()

	for range 3 {
		if err := r.draw(s); err != nil {
			b.Fatal(err)
		}
		if err := r.present(0); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err := r.draw(s); err != nil {
			b.Fatal(err)
		}
		if err := r.present(0); err != nil {
			b.Fatal(err)
		}
	}
}
