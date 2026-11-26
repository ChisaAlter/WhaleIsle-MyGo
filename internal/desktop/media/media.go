// Package media is the process's one image pipeline. Views request a Handle
// by content identity and read its state; reading files, decoding bytes,
// rasterising SVG and decoding animation frames all happen on a small bounded
// worker pool, never in view or build code.
//
// One Service runs per process (Default). Decoded pictures are shared per Key
// in a byte-budgeted LRU, so two views showing the same image share one
// *ui.Bitmap (one texture). Rasterised outputs (SVG, ICO, AVIF stills) are
// also kept on disk under <UserData>/cache/media. Work belongs to an Owner;
// when every owner that asked for a pending picture releases it, the job is
// dropped before it runs or cancelled while it runs.
package media

import (
	"container/list"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo/ui"
)

// State is a Handle's lifecycle.
type State uint8

const (
	Pending State = iota
	Ready
	Failed
)

// Failure classifies a failed picture. It never carries raw decoder text.
type Failure uint8

const (
	FailNone Failure = iota
	// FailUnsupported: a format, or an SVG, that no renderer here draws.
	FailUnsupported
	// FailTooLarge: over the byte or pixel budget.
	FailTooLarge
	// FailCorrupt: the bytes do not decode.
	FailCorrupt
	// FailUnavailable: the source could not be read or fetched.
	FailUnavailable
	// FailCanceled: the owner went away before the picture was made.
	FailCanceled
)

// Error lets a Failure travel as an error. The messages are worded so that
// errcopy classifies them; they are not shown verbatim.
func (f Failure) Error() string {
	switch f {
	case FailUnsupported:
		return "image format is unsupported"
	case FailTooLarge:
		return "image exceeds the preview budget"
	case FailCorrupt:
		return "image data is invalid"
	case FailUnavailable:
		return "image not found"
	case FailCanceled:
		return "image request: context canceled"
	}
	return ""
}

// Key is a picture's identity: what the content is (ID), the pixel size it
// is made for (zero for intrinsic) and the appearance it depends on (for
// SVG currentColor). Two requests with equal Keys share one picture.
type Key struct {
	ID         string
	W, H       int
	Appearance string
	// Cover fits a bitmap to cover W×H (avatars); the default contains it.
	Cover bool
}

// Sized returns k made for a w×h pixel target.
func (k Key) Sized(w, h int) Key { k.W, k.H = max(0, w), max(0, h); return k }

// Picture is a decoded image: a bitmap, or a native vector SVG.
type Picture struct {
	Bitmap        *ui.Bitmap
	SVG           *ui.SVG
	Width, Height float32
	// Extra is what Source.Derive returned with the image, if anything.
	Extra any
}

// Source is the image if the key's picture is not already cached: exactly one
// of Bytes, DataURL, Path, URL or Load. Kind is an optional extension or MIME
// hint (".svg", "image/avif"); content sniffing decides otherwise.
type Source struct {
	Bytes    []byte
	DataURL  string
	Path     string
	URL      string
	Load     func(context.Context) ([]byte, error)
	Kind     string
	MaxBytes int64
	// Derive, for still bitmaps, replaces the decoded (upright) image on the
	// worker and may return derived data for Picture.Extra, such as a color
	// map. The Key must name whatever Derive depends on.
	Derive func(image.Image) (image.Image, any)
	// AutoRev marks a path-keyed source with no caller revision: the service
	// stats the file on a worker and forgets the entry when it changes.
	AutoRev bool
}

type result struct {
	pic  Picture
	anim *animation
	fail Failure
	// rev is the file size+mtime observed at load for AutoRev sources.
	rev string
}

// Handle is one requested picture. Its methods are safe from any goroutine;
// views read it every frame.
type Handle struct {
	svc *Service
	key Key
	src Source
	res atomic.Pointer[result]
	// lastSeen orders the queue: the most recently requested picture is the
	// one on screen now.
	lastSeen atomic.Int64

	// guarded by svc.mu
	owners      map[*Owner]struct{}
	queued      bool
	running     bool
	dropped     bool
	cancel      context.CancelFunc
	lru         *list.Element
	failedElem  *list.Element
	bytes       int64
	failedAt    time.Time
	fileRev     string
	statRunning bool
	statAt      time.Time
}

// State reports whether the picture is pending, ready or failed.
func (h *Handle) State() State {
	if h == nil {
		return Failed
	}
	r := h.res.Load()
	switch {
	case r == nil:
		return Pending
	case r.fail != FailNone:
		return Failed
	}
	return Ready
}

// Picture returns the decoded picture once ready.
func (h *Handle) Picture() (Picture, bool) {
	if h == nil {
		return Picture{}, false
	}
	if r := h.res.Load(); r != nil && r.fail == FailNone {
		return r.pic, true
	}
	return Picture{}, false
}

// Bitmap is the ready bitmap, or nil.
func (h *Handle) Bitmap() *ui.Bitmap {
	p, _ := h.Picture()
	return p.Bitmap
}

// Size is the ready picture's size in pixels (DIPs for vectors).
func (h *Handle) Size() (float32, float32) {
	p, _ := h.Picture()
	return p.Width, p.Height
}

// Failure is the classified failure of a failed handle.
func (h *Handle) Failure() Failure {
	if h == nil {
		return FailUnavailable
	}
	if r := h.res.Load(); r != nil {
		return r.fail
	}
	return FailNone
}

// Err is Failure as an error, nil unless failed.
func (h *Handle) Err() error {
	if f := h.Failure(); f != FailNone {
		return f
	}
	return nil
}

// Animated reports whether the picture plays frames (see Frame).
func (h *Handle) Animated() bool {
	if h == nil {
		return false
	}
	r := h.res.Load()
	return r != nil && r.anim != nil
}

// Frame shows the frame due at now in the picture's one bitmap and returns
// how long until the next frame (zero: still or finished). Call it from the
// view that paints the picture, on the UI thread; a picture nobody paints
// does not advance, so a hidden animation costs nothing.
func (h *Handle) Frame(now time.Time) (*ui.Bitmap, time.Duration) {
	if h == nil {
		return nil, 0
	}
	r := h.res.Load()
	if r == nil || r.fail != FailNone {
		return nil, 0
	}
	if r.anim == nil {
		return r.pic.Bitmap, 0
	}
	return r.pic.Bitmap, r.anim.show(r.pic.Bitmap, now)
}

// Owner is a lease on requests: a page, a tab, a window scope. Releasing it
// (or ending its context) drops every pending picture no other owner wants.
// Ready pictures stay in the shared cache.
type Owner struct {
	svc     *Service
	notify  atomic.Pointer[func()]
	pending map[*Handle]struct{} // guarded by svc.mu
	done    bool                 // guarded by svc.mu
	dirty   atomic.Bool
	stop    func() bool
}

// NewOwner returns a lease whose completions call notify (coalesced). A
// non-nil ctx releases the owner when it ends.
func (s *Service) NewOwner(ctx context.Context, notify func()) *Owner {
	o := &Owner{svc: s, pending: map[*Handle]struct{}{}}
	o.SetNotify(notify)
	if ctx != nil {
		o.stop = context.AfterFunc(ctx, o.Release)
	}
	return o
}

// SetNotify replaces the completion callback (for example the current
// window's Services().Invalidate).
func (o *Owner) SetNotify(notify func()) {
	if o == nil {
		return
	}
	if notify == nil {
		o.notify.Store(nil)
		return
	}
	o.notify.Store(&notify)
}

// Released reports whether the owner was released.
func (o *Owner) Released() bool {
	if o == nil {
		return true
	}
	o.svc.mu.Lock()
	defer o.svc.mu.Unlock()
	return o.done
}

// Release ends the lease. Pending work only this owner wanted never runs;
// running work is cancelled and its result discarded.
func (o *Owner) Release() {
	if o == nil {
		return
	}
	s := o.svc
	s.mu.Lock()
	if o.done {
		s.mu.Unlock()
		return
	}
	o.done = true
	for h := range o.pending {
		delete(h.owners, o)
		if len(h.owners) == 0 {
			s.dropLocked(h)
		}
	}
	clear(o.pending)
	s.mu.Unlock()
	if o.stop != nil {
		o.stop()
	}
}

// changed coalesces completion notifications into one callback.
func (o *Owner) changed() {
	if !o.dirty.CompareAndSwap(false, true) {
		return
	}
	time.AfterFunc(4*time.Millisecond, func() {
		o.dirty.Store(false)
		if f := o.notify.Load(); f != nil && *f != nil {
			(*f)()
		}
	})
}

// Config tunes a Service.
type Config struct {
	// CacheDir holds rasterised outputs; empty disables the disk cache.
	CacheDir string
	// DiskBudget caps CacheDir (default 256 MiB).
	DiskBudget int64
	// MemoryBudget caps decoded pictures (default 192 MiB).
	MemoryBudget int64
	// Workers is the decode pool size (default 2). Negative runs no worker:
	// tests drive the queue with RunPending.
	Workers int
	// Fetch reads a remote URL into a local file (imgcache.Cache.Fetch).
	Fetch func(context.Context, string) (string, error)
}

const (
	defaultMemoryBudget = 192 << 20
	defaultDiskBudget   = 256 << 20
	failureTTL          = 15 * time.Second
	maxFailedEntries    = 256
	workerIdle          = 15 * time.Second
)

// Service decodes pictures for the process.
type Service struct {
	cfg  Config
	disk *diskCache

	mu        sync.Mutex
	entries   map[Key]*Handle
	queue     []*Handle
	lru       list.List
	failedLRU list.List
	bytes     int64
	workers   int
	wake      chan struct{}

	decodes atomic.Int64
	rasters atomic.Int64
}

// New starts an empty service; workers start on demand.
func New(cfg Config) *Service {
	if cfg.MemoryBudget <= 0 {
		cfg.MemoryBudget = defaultMemoryBudget
	}
	if cfg.DiskBudget <= 0 {
		cfg.DiskBudget = defaultDiskBudget
	}
	if cfg.Workers == 0 {
		cfg.Workers = 2
	}
	s := &Service{cfg: cfg, entries: map[Key]*Handle{}, wake: make(chan struct{}, 1)}
	if cfg.CacheDir != "" {
		s.disk = newDiskCache(cfg.CacheDir, cfg.DiskBudget)
	}
	return s
}

var (
	defaultMu  sync.Mutex
	defaultSvc *Service
)

// Init configures the process service. Call it once at startup, before the
// first Default; a later call replaces the service for new requests only.
func Init(cfg Config) *Service {
	s := New(cfg)
	defaultMu.Lock()
	defaultSvc = s
	defaultMu.Unlock()
	return s
}

// Default is the process service (memory-only until Init).
func Default() *Service {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	if defaultSvc == nil {
		defaultSvc = New(Config{})
	}
	return defaultSvc
}

// Decodes counts decodes the service has run (not cache hits). Tests use it
// to prove views decode nothing themselves.
func (s *Service) Decodes() int64 { return s.decodes.Load() }

// Rasters counts resvg renders (disk-cache misses).
func (s *Service) Rasters() int64 { return s.rasters.Load() }

// MemoryBytes is the decoded bytes the LRU holds.
func (s *Service) MemoryBytes() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.bytes
}

// Request returns the handle for key, starting its work if nothing has it
// yet. It never blocks on I/O or decoding: views call it every frame. o may be
// nil for process-lifetime pictures (no cancellation).
func (s *Service) Request(o *Owner, key Key, src Source) *Handle {
	key = normalizeKey(key)
	now := time.Now().UnixNano()
	s.mu.Lock()
	defer s.mu.Unlock()
	if h := s.entries[key]; h != nil {
		r := h.res.Load()
		if r == nil || r.fail == FailNone || time.Since(h.failedAt) < failureTTL {
			h.lastSeen.Store(now)
			if r == nil {
				s.ownLocked(h, o)
			} else if h.lru != nil {
				s.lru.MoveToFront(h.lru)
				s.maybeRevalidateLocked(h)
			}
			return h
		}
		s.removeLocked(h)
	}
	h := &Handle{svc: s, key: key, src: src, owners: map[*Owner]struct{}{}}
	h.lastSeen.Store(now)
	if o != nil && o.done {
		// A released owner asks for nothing new.
		h.res.Store(&result{fail: FailCanceled})
		return h
	}
	s.entries[key] = h
	s.ownLocked(h, o)
	s.enqueueLocked(h)
	return h
}

// fetchSlots bounds concurrent downloads. Downloads never occupy a decode
// worker: a slow network must not hold local pictures back.
var fetchSlots = make(chan struct{}, 4)

func (s *Service) enqueueLocked(h *Handle) {
	if h.src.URL != "" && s.cfg.Fetch != nil {
		ctx, cancel := context.WithCancel(context.Background())
		h.cancel = cancel
		src := h.src
		go func() {
			select {
			case fetchSlots <- struct{}{}:
			case <-ctx.Done():
				cancel()
				return
			}
			var path string
			err := ctx.Err()
			if err == nil {
				path, err = s.cfg.Fetch(ctx, src.URL)
			}
			<-fetchSlots
			cancel()
			s.mu.Lock()
			h.cancel = nil
			if h.dropped {
				s.mu.Unlock()
				return
			}
			if err != nil {
				s.mu.Unlock()
				s.finish(h, &result{fail: classifyLoad(err)})
				return
			}
			h.src = Source{Path: path, Kind: src.Kind, MaxBytes: src.MaxBytes, Derive: src.Derive}
			h.queued = true
			s.queue = append(s.queue, h)
			s.startLocked()
			s.mu.Unlock()
		}()
		return
	}
	h.queued = true
	s.queue = append(s.queue, h)
	s.startLocked()
}

// fileRevalidate is the minimum gap between background stat checks of one
// path-keyed entry; requests per frame must not stat per frame.
const fileRevalidate = time.Second

// statRev is a file's content revision: size and modification time.
func statRev(path string) string {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return ""
	}
	return fmt.Sprintf("%d:%d", info.Size(), info.ModTime().UnixNano())
}

// maybeRevalidateLocked schedules a background stat for a cached path-keyed
// entry whose caller gave no revision. A changed stat removes the stale
// entry; the caller keeps its current picture this frame and the next
// Request decodes the new file. A failed stat keeps the cached picture.
func (s *Service) maybeRevalidateLocked(h *Handle) {
	if !h.src.AutoRev || h.statRunning || time.Since(h.statAt) < fileRevalidate {
		return
	}
	h.statAt = time.Now()
	if s.cfg.Workers < 0 {
		// Synchronous for RunPending-driven tests.
		if rev := statRev(h.src.Path); rev != "" && h.fileRev != "" && rev != h.fileRev {
			s.removeLocked(h)
		}
		return
	}
	h.statRunning = true
	path := h.src.Path
	go func() {
		rev := statRev(path)
		s.mu.Lock()
		h.statRunning = false
		if rev != "" && h.fileRev != "" && rev != h.fileRev {
			s.removeLocked(h)
		}
		s.mu.Unlock()
	}()
}

// Forget drops a cached or failed picture so the next Request retries it.
func (s *Service) Forget(key Key) {
	key = normalizeKey(key)
	s.mu.Lock()
	defer s.mu.Unlock()
	if h := s.entries[key]; h != nil && h.res.Load() != nil {
		s.removeLocked(h)
	}
}

// ForgetID drops every cached or failed picture of a content ID.
func (s *Service) ForgetID(id string) {
	id = normalizeKeyID(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, h := range s.entries {
		if k.ID == id && h.res.Load() != nil {
			s.removeLocked(h)
		}
	}
}

func (s *Service) ownLocked(h *Handle, o *Owner) {
	if o == nil {
		o = s.processOwnerLocked()
	}
	if o.done {
		return
	}
	h.owners[o] = struct{}{}
	o.pending[h] = struct{}{}
}

var processOwners sync.Map // *Service -> *Owner

func (s *Service) processOwnerLocked() *Owner {
	if v, ok := processOwners.Load(s); ok {
		return v.(*Owner)
	}
	o := &Owner{svc: s, pending: map[*Handle]struct{}{}}
	processOwners.Store(s, o)
	return o
}

// SetProcessNotify sets the callback for nil-owner requests.
func (s *Service) SetProcessNotify(notify func()) {
	s.mu.Lock()
	o := s.processOwnerLocked()
	s.mu.Unlock()
	o.SetNotify(notify)
}

// dropLocked retires pending work nobody wants: queued jobs never run, and
// a running job is cancelled with its result discarded.
func (s *Service) dropLocked(h *Handle) {
	if h.res.Load() != nil {
		return
	}
	h.dropped = true
	if h.queued {
		h.queued = false
		for i, q := range s.queue {
			if q == h {
				s.queue = append(s.queue[:i], s.queue[i+1:]...)
				break
			}
		}
	}
	if h.cancel != nil {
		h.cancel()
	}
	if s.entries[h.key] == h {
		delete(s.entries, h.key)
	}
}

func (s *Service) removeLocked(h *Handle) {
	if s.entries[h.key] == h {
		delete(s.entries, h.key)
	}
	if h.lru != nil {
		s.lru.Remove(h.lru)
		h.lru = nil
		s.bytes -= h.bytes
	}
	if h.failedElem != nil {
		s.failedLRU.Remove(h.failedElem)
		h.failedElem = nil
	}
}

func (s *Service) startLocked() {
	if s.cfg.Workers < 0 {
		return
	}
	if s.workers < s.cfg.Workers && s.workers < len(s.queue) {
		s.workers++
		go s.work()
		return
	}
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// nextLocked takes the most recently requested queued job: what is on
// screen now goes first.
func (s *Service) nextLocked() *Handle {
	best := -1
	for i, h := range s.queue {
		if best < 0 || h.lastSeen.Load() > s.queue[best].lastSeen.Load() {
			best = i
		}
	}
	if best < 0 {
		return nil
	}
	h := s.queue[best]
	s.queue = append(s.queue[:best], s.queue[best+1:]...)
	h.queued = false
	return h
}

func (s *Service) work() {
	w := newWorker(s)
	defer w.close()
	idle := time.NewTimer(workerIdle)
	defer idle.Stop()
	for {
		s.mu.Lock()
		h := s.nextLocked()
		if h == nil {
			s.mu.Unlock()
			if !idle.Stop() {
				select {
				case <-idle.C:
				default:
				}
			}
			idle.Reset(workerIdle)
			select {
			case <-s.wake:
				continue
			case <-idle.C:
				s.mu.Lock()
				if len(s.queue) == 0 {
					s.workers--
					s.mu.Unlock()
					return
				}
				s.mu.Unlock()
				continue
			}
		}
		ctx, cancel := context.WithCancel(context.Background())
		h.cancel, h.running = cancel, true
		s.mu.Unlock()
		r := w.run(ctx, h)
		cancel()
		s.finish(h, r)
	}
}

// RunPending runs queued work on the calling goroutine (Workers < 0).
func (s *Service) RunPending() int {
	w := newWorker(s)
	defer w.close()
	n := 0
	for {
		s.mu.Lock()
		h := s.nextLocked()
		if h == nil {
			s.mu.Unlock()
			return n
		}
		ctx, cancel := context.WithCancel(context.Background())
		h.cancel, h.running = cancel, true
		s.mu.Unlock()
		r := w.run(ctx, h)
		cancel()
		s.finish(h, r)
		n++
	}
}

// Queued reports the number of jobs waiting to run.
func (s *Service) Queued() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.queue)
}

func (s *Service) finish(h *Handle, r *result) {
	s.mu.Lock()
	h.running, h.cancel = false, nil
	if h.dropped || r == nil {
		s.mu.Unlock()
		return
	}
	owners := make([]*Owner, 0, len(h.owners))
	for o := range h.owners {
		delete(o.pending, h)
		owners = append(owners, o)
	}
	clear(h.owners)
	h.res.Store(r)
	if r.rev != "" {
		h.fileRev = r.rev
	}
	// Release raw source buffers/closures after decode completes so multi-MB
	// DataURLs and byte slices are not retained alongside the decoded bitmap.
	h.src.Bytes = nil
	h.src.DataURL = ""
	h.src.Load = nil
	h.src.Derive = nil
	if r.fail != FailNone {
		h.failedAt = time.Now()
		if s.entries[h.key] == h {
			if h.failedElem != nil {
				s.failedLRU.MoveToFront(h.failedElem)
			} else {
				h.failedElem = s.failedLRU.PushFront(h)
			}
			s.evictFailedLocked()
		}
	} else if s.entries[h.key] == h {
		h.bytes = r.bytes()
		h.lru = s.lru.PushFront(h)
		s.bytes += h.bytes
		s.evictLocked()
	}
	s.mu.Unlock()
	for _, o := range owners {
		o.changed()
	}
}

func (s *Service) evictFailedLocked() {
	now := time.Now()
	for s.failedLRU.Len() > 0 {
		back := s.failedLRU.Back()
		h := back.Value.(*Handle)
		if s.failedLRU.Len() > maxFailedEntries || now.Sub(h.failedAt) >= failureTTL {
			s.removeLocked(h)
			continue
		}
		break
	}
}

// evictLocked keeps the LRU within its budget. A view still holding an
// evicted picture keeps drawing it; the next Request decodes it again.
func (s *Service) evictLocked() {
	for s.bytes > s.cfg.MemoryBudget && s.lru.Len() > 1 {
		back := s.lru.Back()
		h := back.Value.(*Handle)
		s.removeLocked(h)
	}
}

func (r *result) bytes() int64 {
	if r.anim != nil {
		return r.anim.bytes
	}
	if r.pic.Bitmap != nil {
		w, h := r.pic.Bitmap.Size()
		return int64(w) * int64(h) * 4
	}
	if r.pic.SVG != nil {
		return 64 << 10
	}
	return 0
}

// Image returns the picture as a ui.Image source, nil if empty.
func (p Picture) Image() ui.ImageSource {
	if p.Bitmap != nil {
		return p.Bitmap
	}
	if p.SVG != nil {
		return p.SVG
	}
	return nil
}

func hashDataURL(s string) string {
	sum := sha256.Sum256([]byte(s))
	return "dataurl:" + hex.EncodeToString(sum[:])
}

func normalizeKeyID(id string) string {
	if strings.HasPrefix(id, "data:") {
		return hashDataURL(id)
	}
	return id
}

func normalizeKey(key Key) Key {
	key.ID = normalizeKeyID(key.ID)
	return key
}

// DataURL keys a data: URL by its SHA-256 digest so multi-MB payloads are not
// retained as map keys; the base64 or percent decoding happens on a worker.
func DataURL(s string) (Key, Source) {
	return Key{ID: hashDataURL(s)}, Source{DataURL: s}
}

// File keys a local file by its path; it is read on a worker. rev is a
// content revision the caller already knows without doing UI-thread I/O
// (size+mtime from a listing, a config value). A changed rev is a new
// picture. With rev == "" the key is the bare path and the service
// revalidates the file's size/mtime on a worker in the background; a changed
// stat forgets the stale entry so the next request decodes the new bytes.
func File(path, rev string) (Key, Source) {
	src := Source{Path: path}
	if rev != "" {
		return Key{ID: "file:" + path + "#" + rev}, src
	}
	src.AutoRev = true
	return Key{ID: "file:" + path}, src
}

// Remote keys a public http(s) image; it is fetched through the configured
// image cache on a worker.
func Remote(rawURL string) (Key, Source) { return Key{ID: "url:" + rawURL}, Source{URL: rawURL} }

// Bytes keys in-memory bytes by their SHA-256. Hash once, not per frame:
// callers keep the Key with the bytes.
func Bytes(data []byte, kind string) (Key, Source) {
	sum := sha256.Sum256(data)
	return Key{ID: "sha256:" + hex.EncodeToString(sum[:])}, Source{Bytes: data, Kind: kind}
}

// Cached reports whether key has a ready picture in the cache, without
// requesting it.
func (s *Service) Cached(key Key) bool {
	key = normalizeKey(key)
	s.mu.Lock()
	defer s.mu.Unlock()
	h := s.entries[key]
	return h != nil && h.lru != nil
}
