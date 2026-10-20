package session

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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
			return nil, errors.New(fmt.Sprintf(storelocale.T(storelocale.KeyStoreSessionChildImagesOnly), filepath.Base(path)))
		}
		block, _ := json.Marshal(map[string]string{"type": "image", "mediaType": mediaType, "data": base64.StdEncoding.EncodeToString(data), "name": filepath.Base(path)})
		parts = append(parts, block)
	}
	return parts, nil
}

func (s *Store) AddAgentNavigationImages(token uint64, address AgentSessionAddress, paths []string, pickErr error) {
	s.mu.Lock()
	if !s.agentPromptActionAdmissionLocked(token, address, true) || s.agentNavigation.view.Address != address || !s.agentParentCanPromptLocked(address) || s.agentNavigation.view.ActionBusy {
		s.mu.Unlock()
		return
	}
	if pickErr != nil {
		s.agentNavigation.view.ActionStatus = pickErr.Error()
	} else {
		combined := append([]string(nil), s.agentNavigation.view.PromptImages...)
		for _, path := range paths {
			if path != "" && !slices.Contains(combined, path) {
				combined = append(combined, path)
			}
		}
		if err := validateChildImages(combined, s.agentNavigation.view.ImageLimits); err != nil {
			s.agentNavigation.view.ActionStatus = err.Error()
			s.mu.Unlock()
			s.fire()
			return
		}
		for _, path := range paths {
			if path == "" {
				continue
			}
			found := false
			for _, old := range s.agentNavigation.view.PromptImages {
				if old == path {
					found = true
					break
				}
			}
			if !found {
				s.agentNavigation.view.PromptImages = append(s.agentNavigation.view.PromptImages, path)
			}
		}
	}
	s.mu.Unlock()
	s.fire()
}

func (s *Store) RemoveAgentNavigationImage(token uint64, address AgentSessionAddress, path string) {
	s.mu.Lock()
	if !s.agentPromptActionAdmissionLocked(token, address, true) || s.agentNavigation.view.Address != address || s.agentNavigation.view.ActionBusy {
		s.mu.Unlock()
		return
	}
	old := s.agentNavigation.view.PromptImages
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
