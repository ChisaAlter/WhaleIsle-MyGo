package main

import (
	"context"
	"fmt"
	"github.com/egoist/mygo"
	"time"
	"whaleisle/internal/projectflow"
)

func (s *mainState) runProjectAsset(win *mygo.Window, id, action string, file projectflow.Artifact, workID string, open func(string)) {
	if win == nil || id == "" {
		return
	}
	s.mu.Lock()
	f := &s.projectFlow
	client := s.dshClient
	if f.busy {
		s.mu.Unlock()
		return
	}
	if client == nil || f.client != client || !f.listing.Available || f.requestProjectID != id || f.requestCard == nil {
		f.err = resumeText("请先刷新交付详情。", "Refresh the delivery details first.")
		s.mu.Unlock()
		return
	}
	allowed := action == "directory"
	if action == "file" && open != nil && f.requestCard.Delivery != nil {
		for _, candidate := range f.requestCard.Delivery.Artifacts {
			if candidate.Path == file.Path && candidate.WorkstreamID == file.WorkstreamID && candidate.DelegationRef == file.DelegationRef {
				allowed = true
				break
			}
		}
	}
	if action == "diff" {
		for _, work := range f.requestCard.Work {
			if work.WorkstreamID == workID {
				allowed = true
				break
			}
		}
	}
	if !allowed {
		f.err = resumeText("所选文件或工作不属于当前交付引用。", "Selected file or work does not belong to the displayed delivery references.")
		s.mu.Unlock()
		return
	}
	requestID := f.requestCard.ID
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
			f := &s.projectFlow
			if s.dshClient != client || f.client != client || f.generation != generation {
				s.mu.Unlock()
				return
			}
			f.busy = false
			if f.requestProjectID != id || f.requestGeneration != requestGeneration || f.requestCard == nil || f.requestCard.ID != requestID {
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
				if !f.expanded || f.navigation != navigation || sessions == nil || sessions.Snapshot().SelectedID != selected {
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
