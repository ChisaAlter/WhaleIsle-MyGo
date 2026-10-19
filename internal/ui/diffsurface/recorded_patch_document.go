package diffsurface

// RecordedPatchFile and RecordedPatchRow expose the existing native patch
// document to other read-only snapshot surfaces, without repository actions.
type RecordedPatchFile = patchFile
type RecordedPatchRow = patchRow

func ParseRecordedPatch(source string) []RecordedPatchFile {
	return parsePatchFiles(source)
}
