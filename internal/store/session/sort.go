package session

import "sort"

// PinRank returns the pin order index for id, or -1 when not pinned.
func PinRank(id string, pinnedOrder []string) int {
	for i, pid := range pinnedOrder {
		if pid == id {
			return i
		}
	}
	return -1
}

// SortItemsSidebar orders roots/children: pinned first (pin order), then updatedAt desc.
func SortItemsSidebar(items []Item, pinnedOrder []string) {
	sort.Slice(items, func(i, j int) bool {
		a, b := items[i], items[j]
		ra, rb := PinRank(a.SessionID, pinnedOrder), PinRank(b.SessionID, pinnedOrder)
		if ra >= 0 && rb < 0 {
			return true
		}
		if ra < 0 && rb >= 0 {
			return false
		}
		if ra >= 0 && rb >= 0 && ra != rb {
			return ra < rb
		}
		return a.UpdatedAt > b.UpdatedAt
	})
}
