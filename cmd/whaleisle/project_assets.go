package main

import (
	"context"
	"fmt"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/projectflow"
)

func (s *projectController) runProjectAsset(win *mygo.Window, id, action string, file projectflow.Artifact, workID string, open func(string), owners ...projectQueueSubmission) {
	if win == nil || id == "" {
		return
	}
	s.mu.Lock()
	f := s.projectFlow
	client := s.dshClient
	if len(owners) > 0 && !owners[0].matches(f, client) {
		s.mu.Unlock()
		return
	}
	if f.busy {
		s.mu.Unlock()
		return
	}
	if client == nil || f.client != client || !f.listing.Available {
		f.err = resumeText("请先刷新交付详情。", "Refresh the delivery details first.")
		s.mu.Unlock()
		return
	}
	requestID, admissionErr := projectAssetReference(f, id, action, file, workID, open != nil)
	if admissionErr != nil {
		f.err = admissionErr.Error()
		s.mu.Unlock()
		return
	}
	requestGeneration := f.requestGeneration
	navigation := f.navigation
	sessions := s.sessions
	f.busy = true
	f.err = ""
	f.assetNote = ""
	f.generation++
	generation := f.generation
	s.mu.Unlock()
	selected := ""
	if sessions != nil {
		selected = sessions.Snapshot().SelectedID
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		rpc := projectflow.Client{Origin: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTP: client.HTTP}
		var resolved projectflow.ResolvedArtifact
		var diff projectflow.WorkDiff
		var err error
		switch action {
		case "file":
			resolved, err = rpc.ResolveArtifact(ctx, id, file)
		case "diff":
			diff, err = rpc.WorkDiff(ctx, id, workID)
		case "directory":
			err = rpc.OpenProjectDirectory(ctx, id)
		default:
			err = fmt.Errorf("unsupported Project artifact action")
		}
		win.Update(func() {
			s.mu.Lock()
			f := s.projectFlow
			if s.dshClient != client || f.client != client || f.generation != generation {
				s.mu.Unlock()
				return
			}
			f.busy = false
			if action != "directory" && (f.requestProjectID != id || f.requestGeneration != requestGeneration || f.requestCard == nil || f.requestCard.ID != requestID) {
				s.mu.Unlock()
				return
			}
			if err != nil {
				f.err = err.Error()
				s.mu.Unlock()
				return
			}
			switch action {
			case "directory":
				f.assetNote = resumeText("已请求打开项目目录。", "Requested opening the Project directory.")
			case "diff":
				f.assetDiff = &diff
			case "file":
				if !f.visible() || f.navigation != navigation || sessions == nil || sessions.Snapshot().SelectedID != selected {
					f.err = resumeText("导航已改变，未打开文件。", "Navigation changed; the file was not opened.")
					s.mu.Unlock()
					return
				}
				if resolved.ModifiedSinceReport {
					f.assetNote = resumeText("此文件在报告后已修改，打开的是当前内容。", "This file changed after the report; opening its current contents.")
				}
				// Route through the same native presented-file entry as chat deliverables.
				// Its dirty-editor guard owns admission; never bypass it with an OS opener.
				s.mu.Unlock()
				open(resolved.CanonicalPath)
				return
			}
			s.mu.Unlock()
		})
	}()
}

func projectAssetReference(f *projectFlowState, id, action string, file projectflow.Artifact, workID string, canOpen bool) (string, error) {
	if action == "directory" {
		for _, project := range f.listing.Projects {
			if project.ID == id {
				return "", nil
			}
		}
		return "", fmt.Errorf("Project is no longer in the current list; refresh first")
	}
	if f.requestProjectID != id || f.requestCard == nil {
		return "", fmt.Errorf("Refresh the delivery details first")
	}
	if action == "file" && canOpen && f.requestCard.Delivery != nil {
		for _, candidate := range f.requestCard.Delivery.Artifacts {
			if candidate.Path == file.Path && candidate.WorkstreamID == file.WorkstreamID && candidate.DelegationRef == file.DelegationRef {
				return f.requestCard.ID, nil
			}
		}
	}
	if action == "diff" {
		for _, work := range f.requestCard.Work {
			if work.WorkstreamID == workID {
				return f.requestCard.ID, nil
			}
		}
	}
	return "", fmt.Errorf("Selected file or work does not belong to the displayed delivery references")
}
