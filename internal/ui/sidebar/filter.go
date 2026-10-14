package sidebar

import (
	"strings"

	"whaleisle/internal/store/session"
)

func sidebarItems(snap session.ViewState, queryPtr *string) []session.Item {
	archived := make(map[string]bool, len(snap.ArchivedSessionIDs))
	for _, id := range snap.ArchivedSessionIDs {
		archived[id] = true
	}
	items := make([]session.Item, 0, len(snap.Items))
	for _, it := range snap.Items {
		if !archived[it.SessionID] {
			items = append(items, it)
		}
	}
	if queryPtr == nil {
		return items
	}
	q := strings.TrimSpace(*queryPtr)
	if q == "" {
		return items
	}
	rpcReady := !snap.SearchLoading &&
		!snap.SearchUseLocal &&
		snap.SearchErr == "" &&
		snap.SearchRPCQuery == q
	if rpcReady {
		return filterItemsByHitIDs(items, snap.SearchHitIDs)
	}
	return filterItems(items, q)
}

func filterItemsByHitIDs(items []session.Item, hits map[string]bool) []session.Item {
	if len(hits) == 0 {
		return nil
	}
	byID := make(map[string]session.Item, len(items))
	children := make(map[string][]string)
	for _, it := range items {
		byID[it.SessionID] = it
		if it.ParentID != "" {
			children[it.ParentID] = append(children[it.ParentID], it.SessionID)
		}
	}
	visible := map[string]bool{}
	for id, ok := range hits {
		if !ok {
			continue
		}
		it, ok := byID[id]
		if !ok {
			continue
		}
		visible[it.SessionID] = true
		for pid := it.ParentID; pid != ""; {
			visible[pid] = true
			pid = byID[pid].ParentID
		}
		var markDesc func(id string)
		markDesc = func(id string) {
			for _, cid := range children[id] {
				visible[cid] = true
				markDesc(cid)
			}
		}
		markDesc(it.SessionID)
	}
	out := make([]session.Item, 0, len(visible))
	for _, it := range items {
		if visible[it.SessionID] {
			out = append(out, it)
		}
	}
	return out
}

func filterItems(items []session.Item, query string) []session.Item {
	q := strings.TrimSpace(strings.ToLower(query))
	if q == "" {
		return items
	}
	byID := make(map[string]session.Item, len(items))
	children := make(map[string][]string)
	for _, it := range items {
		byID[it.SessionID] = it
		if it.ParentID != "" {
			children[it.ParentID] = append(children[it.ParentID], it.SessionID)
		}
	}
	match := func(it session.Item) bool {
		return strings.Contains(strings.ToLower(it.Title), q) ||
			strings.Contains(strings.ToLower(it.SessionID), q)
	}
	visible := map[string]bool{}
	for _, it := range items {
		if !match(it) {
			continue
		}
		visible[it.SessionID] = true
		for pid := it.ParentID; pid != ""; {
			visible[pid] = true
			pid = byID[pid].ParentID
		}
		var markDesc func(id string)
		markDesc = func(id string) {
			for _, cid := range children[id] {
				visible[cid] = true
				markDesc(cid)
			}
		}
		markDesc(it.SessionID)
	}
	out := make([]session.Item, 0, len(visible))
	for _, it := range items {
		if visible[it.SessionID] {
			out = append(out, it)
		}
	}
	return out
}
