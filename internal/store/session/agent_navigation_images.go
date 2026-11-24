package session

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"whaleisle/internal/store/storelocale"
)

// Child continuation accepts image data directly; files must never be staged
// through fileUploads or sent through the ordinary session prompt route.
func agentNavigationPromptContent(text string, paths []string) ([]json.RawMessage, error) {
	return agentNavigationPromptContentWithLimits(text, paths, nil)
}
func agentNavigationPromptContentWithLimits(text string, paths []string, limits *ComposerImageLimits) ([]json.RawMessage, error) {
	if err := validateChildImages(paths, limits); err != nil {
		return nil, err
	}
	parts := PromptContentBlocks(text)
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		mediaType := http.DetectContentType(data)
		switch mediaType {
		case "image/png", "image/jpeg", "image/gif", "image/webp":
		default:
			return nil, storelocale.Errorf(storelocale.KeyStoreSessionChildImagesOnly, filepath.Base(path))
		}
		block, _ := json.Marshal(map[string]string{"type": "image", "mediaType": mediaType, "data": base64.StdEncoding.EncodeToString(data), "name": filepath.Base(path)})
		parts = append(parts, block)
	}
	return parts, nil
}

var childImageWorkers = make(chan struct{}, 2)

// AddAgentNavigationImages retains the synchronous non-UI contract. Native
// input callers use AddAgentNavigationImagesAsync so disk I/O never runs in an
// input callback. Both share the same target, validation and completion owner.
func (s *Store) AddAgentNavigationImages(token uint64, address AgentSessionAddress, paths []string, pickErr error) {
	s.addAgentNavigationImages(token, address, paths, pickErr, nil, false)
}

func (s *Store) AddAgentNavigationImagesAsync(token uint64, address AgentSessionAddress, paths []string, pickErr error) {
	s.addAgentNavigationImages(token, address, paths, pickErr, nil, true)
}

// PrepareAgentNavigationImagesAsync admits the image input before performing
// clipboard decoding/storage, so Send cannot race the preparatory file write.
func (s *Store) PrepareAgentNavigationImagesAsync(token uint64, address AgentSessionAddress, prepare func() ([]string, error)) {
	if prepare == nil {
		return
	}
	s.addAgentNavigationImages(token, address, nil, nil, prepare, true)
}

func (s *Store) addAgentNavigationImages(token uint64, address AgentSessionAddress, paths []string, pickErr error, prepare func() ([]string, error), async bool) {
	s.mu.Lock()
	if !s.agentPromptActionAdmissionLocked(token, address, true) || s.agentNavigation.view.Address != address || !s.agentParentCanPromptLocked(address) || s.agentNavigation.view.ActionBusy || s.agentNavigation.view.ImagesPreparing {
		s.mu.Unlock()
		return
	}
	if pickErr != nil {
		s.agentNavigation.view.ActionStatus = faultText("agentNavigation.pickImages", pickErr)
		s.invalidateViewLocked()
		s.mu.Unlock()
		s.fire()
		return
	}
	combined := append([]string(nil), s.agentNavigation.view.PromptImages...)
	for _, path := range paths {
		if path != "" && !slices.Contains(combined, path) {
			combined = append(combined, path)
		}
	}
	if prepare == nil && len(combined) == len(s.agentNavigation.view.PromptImages) {
		s.mu.Unlock()
		return
	}
	limits := cloneChildImageLimits(s.agentNavigation.view.ImageLimits)
	s.agentNavigation.imageEpoch++
	epoch, generation, client := s.agentNavigation.imageEpoch, s.agentNavigation.generation, s.agentNavigation.client
	ctx := s.agentReadContextLocked()
	s.agentNavigation.view.ImagesPreparing = true
	s.mu.Unlock()
	s.fire()
	work := func() {
		current := func() bool {
			return s.agentNavigation.view.Token == token && s.agentNavigation.view.Address == address &&
				s.agentNavigation.imageEpoch == epoch && s.agentNavigation.generation == generation && s.agentNavigation.client == client
		}
		s.mu.Lock()
		valid := current()
		s.mu.Unlock()
		if !valid {
			return
		}
		err := runChildImageWorker(ctx, func() error {
			if prepare != nil {
				paths, err := prepare()
				if err != nil {
					return err
				}
				for _, path := range paths {
					if path != "" && !slices.Contains(combined, path) {
						combined = append(combined, path)
					}
				}
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			return validateChildImages(combined, limits)
		})
		s.mu.Lock()
		if !current() {
			s.mu.Unlock()
			return
		}
		s.agentNavigation.view.ImagesPreparing = false
		if err != nil {
			s.agentNavigation.view.ActionStatus = faultText("agentNavigation.images", err)
		} else if s.agentParentCanPromptLocked(address) {
			s.agentNavigation.view.PromptImages = combined
			s.agentNavigation.view.ActionStatus = ""
		}
		s.invalidateViewLocked()
		s.mu.Unlock()
		s.fire()
	}
	if async {
		go work()
	} else {
		work()
	}
}

func runChildImageWorker(ctx context.Context, work func() error) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case childImageWorkers <- struct{}{}:
	}
	defer func() { <-childImageWorkers }()
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := work(); err != nil {
		return err
	}
	return ctx.Err()
}

func (s *Store) RemoveAgentNavigationImage(token uint64, address AgentSessionAddress, path string) {
	s.mu.Lock()
	if !s.agentPromptActionAdmissionLocked(token, address, true) || s.agentNavigation.view.Address != address || s.agentNavigation.view.ActionBusy {
		s.mu.Unlock()
		return
	}
	old := s.agentNavigation.view.PromptImages
	s.agentNavigation.imageEpoch++
	s.agentNavigation.view.ImagesPreparing = false
	kept := make([]string, 0, len(old))
	for _, p := range old {
		if p != path {
			kept = append(kept, p)
		}
	}
	s.agentNavigation.view.PromptImages = kept
	s.mu.Unlock()
	s.fire()
}
