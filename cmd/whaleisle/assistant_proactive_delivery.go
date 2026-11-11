package main

import (
	"context"
	"time"
)

func (s *mainState) refreshProactiveDeliveryRoutes() {
	s.mu.Lock()
	runner := s.proactivePanel.runner
	if runner == nil || s.proactivePanel.deliveryBusy {
		s.mu.Unlock()
		return
	}
	s.proactivePanel.deliveryBusy = true
	s.proactivePanel.deliveryError = ""
	s.mu.Unlock()
	s.refreshWhaleSettingsUI()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		routes, err := runner.DeliveryRoutes(ctx)
		s.mu.Lock()
		if s.proactivePanel.runner == runner {
			s.proactivePanel.deliveryBusy = false
			s.proactivePanel.deliveryRoutes = routes
			if err != nil {
				s.proactivePanel.deliveryError = err.Error()
			}
		}
		s.mu.Unlock()
		s.refreshWhaleSettingsUI()
	}()
}
