package main

import "whaleisle/internal/projectflow"

func projectRequestCardKey(projectID, requestID string) string {
	return projectID + "\x00" + requestID
}

// rememberProjectRequestCardLocked accepts only a Host result already admitted
// by the caller's controller/request ownership checks. UI card copies never
// populate this cache. The caller holds s.mu.
func (s *projectController) rememberProjectRequestCardLocked(card *projectflow.RequestCard) {
	f := s.projectFlow
	if card == nil || card.ID == "" || card.ProjectID == "" || card.ProjectID != f.requestProjectID || f.client == nil {
		return
	}
	copy := *card
	if card.Changes != nil {
		changes := *card.Changes
		if changes.RevertedAt != nil {
			revertedAt := *changes.RevertedAt
			changes.RevertedAt = &revertedAt
		}
		copy.Changes = &changes
	}
	if card.Delivery != nil {
		delivery := *card.Delivery
		// Page references are values; freeze the slice and delivery container
		// so another card read or a retained UI snapshot cannot change grants.
		delivery.Pages = append([]projectflow.DeliveryPageReference(nil), delivery.Pages...)
		copy.Delivery = &delivery
	}
	if f.requestCards == nil {
		f.requestCards = make(map[string]projectflow.RequestCard)
		f.requestReverts = nil
	}
	f.requestCards[projectRequestCardKey(card.ProjectID, card.ID)] = copy
}
