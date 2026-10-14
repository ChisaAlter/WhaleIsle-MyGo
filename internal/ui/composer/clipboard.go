package composer

// PasteStillCurrent admits a prepared image only to the exact originating
// session draft and edit mode. It cannot follow navigation into another draft.
func PasteStillCurrent(original, current *DraftState, originID, currentID string, editSeq float64) bool {
	return original != nil && original == current && originID == currentID && original.EditSeq == editSeq && !original.Sending
}
