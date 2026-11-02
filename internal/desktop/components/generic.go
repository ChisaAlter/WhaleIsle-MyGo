package components

import (
	"bufio"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"whaleisle/internal/processutil"
	"whaleisle/internal/store/storelocale"
)

//go:embed generic-source/index.js generic-source/store.js generic-source/registry.js generic-source/lifecycle.js generic-source/worker.js
var genericSource embed.FS

// GenericOptions uses the Host's already resolved Node and resource root.
// No PATH/toolchain discovery is performed by this package.
type GenericOptions struct {
	NodeBin     string
	SamplesRoot string
	OnProgress  func(id, action string, progress GenericProgress)
}
type GenericProgress struct {
	ID      string `json:"id"`
	Op      string `json:"op"`
	Phase   string `json:"phase"`
	Percent int    `json:"percent"`
	Message string `json:"message"`
}
type GenericResult struct {
	OK         bool   `json:"ok"`
	Error      string `json:"error"`
	Message    string `json:"message"`
	State      string `json:"state"`
	From       string `json:"from"`
	To         string `json:"to"`
	Components []Row  `json:"components"`
}
type genericResponse struct {
	GenericResult
	RequestID uint64           `json:"requestId"`
	Progress  *GenericProgress `json:"progress"`
}
type genericPending struct {
	result   chan genericResponse
	progress func(GenericProgress)
}
type genericWorker struct {
	mu         sync.Mutex
	writeMu    sync.Mutex
	cmd        *exec.Cmd
	input      io.WriteCloser
	pending    map[uint64]genericPending
	next       uint64
	dead       error
	onProgress func(string, string, GenericProgress)
}
type genericService struct {
	mu      sync.Mutex
	options GenericOptions
	worker  *genericWorker
	closed  bool
}

var genericServices = struct {
	sync.Mutex
	byRoot map[string]*genericService
}{byRoot: map[string]*genericService{}}

// ConfigureGeneric does not start the worker or any component. Call with the
// same resolved resources as the Host before showing the component catalog.
func ConfigureGeneric(userData string, options GenericOptions) {
	root := filepath.Clean(userData)
	genericServices.Lock()
	defer genericServices.Unlock()
	if old := genericServices.byRoot[root]; old != nil {
		old.mu.Lock()
		old.options = options
		old.mu.Unlock()
		return
	}
	genericServices.byRoot[root] = &genericService{options: options}
}
func genericServiceFor(userData string) *genericService {
	genericServices.Lock()
	defer genericServices.Unlock()
	return genericServices.byRoot[filepath.Clean(userData)]
}
func listGeneric(userData string) []Row {
	s := genericServiceFor(userData)
	if s == nil {
		return genericDiskRows(userData, "")
	}
	s.mu.Lock()
	samples := s.options.SamplesRoot
	worker := s.worker
	s.mu.Unlock()
	// An existing worker owns reconciliation and the latest supervised state.
	if worker != nil {
		result, err := worker.request(map[string]any{"op": "list"}, nil)
		if err == nil && result.Components != nil {
			return result.Components
		}
	}
	return genericDiskRows(userData, samples)
}

// RunGenericAction preserves the original service result, including a switched
// version whose restart failed. Callers must display Message/State, not only OK.
func RunGenericAction(userData, id, action string) (GenericResult, error) {
	return runGenericTarget(userData, id, action, id)
}

// InstallGenericVersion selects an existing catalog version; empty selects latest.
func InstallGenericVersion(userData, id, version string) (GenericResult, error) {
	return runGenericTarget(userData, id, "install", map[string]string{"id": id, "version": version})
}

func runGenericTarget(userData, id, action string, target any) (GenericResult, error) {
	if !genericID.MatchString(id) || id == "whalebridge" || id == HostComponentID {
		return GenericResult{}, errors.New("invalid-component-id")
	}
	switch action {
	case "install", "start", "stop", "update", "rollback", "uninstall":
	default:
		return GenericResult{}, ErrUnknownAction
	}
	s := genericServiceFor(userData)
	if s == nil {
		return GenericResult{}, errors.New(storelocale.T(storelocale.KeyStoreComponentsTheComponentRuntimeIsNotConfigured))
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return GenericResult{}, errors.New(storelocale.T(storelocale.KeyStoreComponentsTheComponentServiceIsExiting))
	}
	opts := s.options
	if s.worker != nil {
		s.worker.mu.Lock()
		dead := s.worker.dead != nil
		s.worker.mu.Unlock()
		if dead {
			s.worker = nil
		}
	}
	if s.worker == nil {
		worker, err := startGenericWorker(userData, opts)
		if err != nil {
			s.mu.Unlock()
			return GenericResult{}, err
		}
		s.worker = worker
	}
	worker := s.worker
	s.mu.Unlock()
	result, err := worker.request(map[string]any{"op": action, "target": target}, func(p GenericProgress) {
		if opts.OnProgress != nil {
			opts.OnProgress(id, action, p)
		}
	})
	if err != nil {
		return result, err
	}
	if !result.OK {
		if result.Error == "not-installed" {
			return result, ErrNotInstalled
		}
		return result, fmt.Errorf("%s: %s", result.Error, result.Message)
	}
	if result.State == "error" {
		return result, errors.New(fmt.Sprintf(storelocale.T(storelocale.KeyStoreComponentsTheComponentSwitchedVersionsButFailedT), result.Message))
	}
	return result, nil
}

// ShutdownGeneric belongs only on committed quit, never window close/cancel.
func ShutdownGeneric(userData string) error {
	s := genericServiceFor(userData)
	if s == nil {
		return nil
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	worker := s.worker
	if worker == nil {
		for _, row := range genericDiskRows(userData, s.options.SamplesRoot) {
			if row.State == "running" && row.PID > 0 {
				var err error
				worker, err = startGenericWorker(userData, s.options)
				if err != nil {
					s.mu.Unlock()
					return err
				}
				s.worker = worker
				break
			}
		}
	}
	s.mu.Unlock()
	if worker == nil {
		return nil
	}
	_, err := worker.request(map[string]any{"op": "shutdown"}, nil)
	_ = worker.input.Close()
	return err
}

func startGenericWorker(userData string, opts GenericOptions) (*genericWorker, error) {
	if opts.NodeBin == "" {
		return nil, errors.New(storelocale.T(storelocale.KeyStoreComponentsTheComponentNeedsTheHostNodeRuntimeCh))
	}
	source, err := genericSource.ReadFile("generic-source/worker.js")
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(opts.NodeBin, "-e", string(source))
	processutil.BackgroundTree(cmd)
	cmd.Env = os.Environ()
	input, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		_ = input.Close()
		return nil, err
	}
	cmd.Stderr = io.Discard // Protocol errors are returned as structured results.
	if err = cmd.Start(); err != nil {
		_ = input.Close()
		return nil, err
	}
	worker := &genericWorker{cmd: cmd, input: input, pending: map[uint64]genericPending{}, onProgress: opts.OnProgress}
	go worker.read(output)
	sources := map[string]string{}
	for _, name := range []string{"index", "store", "registry", "lifecycle"} {
		data, e := genericSource.ReadFile("generic-source/" + name + ".js")
		if e != nil {
			_ = input.Close()
			return nil, e
		}
		sources[name] = string(data)
	}
	result, err := worker.request(map[string]any{"op": "initialize", "sources": sources, "userData": userData, "samplesRoot": opts.SamplesRoot}, nil)
	if err != nil || !result.OK {
		_ = input.Close()
		if err == nil {
			err = fmt.Errorf("%s: %s", result.Error, result.Message)
		}
		return nil, err
	}
	return worker, nil
}
func (w *genericWorker) request(request map[string]any, progress func(GenericProgress)) (GenericResult, error) {
	w.mu.Lock()
	if w.dead != nil {
		err := w.dead
		w.mu.Unlock()
		return GenericResult{}, err
	}
	w.next++
	id := w.next
	request["requestId"] = id
	pending := genericPending{result: make(chan genericResponse, 1), progress: progress}
	w.pending[id] = pending
	w.mu.Unlock()
	w.writeMu.Lock()
	err := json.NewEncoder(w.input).Encode(request)
	w.writeMu.Unlock()
	if err != nil {
		w.mu.Lock()
		delete(w.pending, id)
		w.mu.Unlock()
		return GenericResult{}, err
	}
	response := <-pending.result
	if response.Error == "worker-exited" {
		return response.GenericResult, errors.New(response.Message)
	}
	return response.GenericResult, nil
}
func (w *genericWorker) read(output io.Reader) {
	scanner := bufio.NewScanner(output)
	scanner.Buffer(make([]byte, 4096), 4<<20)
	for scanner.Scan() {
		var response genericResponse
		if json.Unmarshal(scanner.Bytes(), &response) != nil {
			continue
		}
		w.mu.Lock()
		pending, ok := w.pending[response.RequestID]
		if response.Progress == nil {
			delete(w.pending, response.RequestID)
		}
		w.mu.Unlock()
		if response.Progress != nil {
			p := *response.Progress
			if ok && pending.progress != nil {
				go pending.progress(p)
			} else if w.onProgress != nil {
				go w.onProgress(p.ID, p.Op, p)
			}
		} else if ok {
			pending.result <- response
		}
	}
	err := w.cmd.Wait()
	if err == nil {
		err = io.EOF
	}
	if scanner.Err() != nil {
		err = scanner.Err()
	}
	w.mu.Lock()
	w.dead = errors.New(fmt.Sprintf(storelocale.T(storelocale.KeyStoreComponentsTheComponentBackgroundExitedW), err))
	for id, pending := range w.pending {
		pending.result <- genericResponse{GenericResult: GenericResult{Error: "worker-exited", Message: w.dead.Error()}}
		delete(w.pending, id)
	}
	w.mu.Unlock()
}
