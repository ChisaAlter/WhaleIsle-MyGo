package main

import (
	"fmt"
	"whaleisle/internal/dsh"
	"whaleisle/internal/projectflow"
)

type projectQueueSubmission struct {
	Change                                    projectflow.RequestQueueChange
	Client                                    *dsh.Client
	Generation, Navigation, RequestGeneration uint64
}

func (o projectQueueSubmission) matches(f *projectFlowState, client *dsh.Client) bool {
	return o.Client != nil && client == o.Client && f.client == o.Client && f.generation == o.Generation && f.navigation == o.Navigation && f.requestGeneration == o.RequestGeneration
}

// The source moves a card before a queued identity; indexes never cross the
// wire. Preserve the revision shown to the user for the Host's atomic check.
func projectQueueChange(page projectflow.RequestPage, detail *projectflow.RequestCard, queue *projectflow.RequestQueue, id, action string) (projectflow.RequestQueueChange, error) {
	change := projectflow.RequestQueueChange{RequestID: id, Action: action}
	if queue == nil {
		return change, fmt.Errorf("Project request queue is unavailable on this Host")
	}
	if detail != nil && detail.ID == id {
		change.RequestRevision = detail.Revision
	}
	for _, item := range page.Items {
		if item.ID == id && change.RequestRevision == 0 {
			change.RequestRevision = item.Revision
			break
		}
	}
	index := -1
	for i, item := range queue.Queued {
		if item.ID == id {
			index = i
			if change.RequestRevision == 0 {
				change.RequestRevision = item.Revision
			}
			break
		}
	}
	if action == "moveUp" || action == "moveDown" {
		if index < 0 {
			return change, fmt.Errorf("Requirement is no longer queued; refresh first")
		}
		change.Action = "move"
		if action == "moveUp" {
			if index == 0 {
				return change, fmt.Errorf("Requirement is already first")
			}
			change.BeforeRequestID = queue.Queued[index-1].ID
		} else {
			if index+1 >= len(queue.Queued) {
				return change, fmt.Errorf("Requirement is already last")
			}
			if index+2 < len(queue.Queued) {
				change.BeforeRequestID = queue.Queued[index+2].ID
			}
		}
	}
	if id == "" || change.RequestRevision <= 0 {
		return change, fmt.Errorf("Refresh the current requirement revision first")
	}
	if change.Action != "enqueue" && change.Action != "move" && change.Action != "cancel" && change.Action != "resume" {
		return change, fmt.Errorf("Unknown request queue action")
	}
	return change, nil
}
