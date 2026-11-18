package storelocale

// Copy for a settings write the Host did not store. A page shows the sentence
// and keeps the Host's own words for the log, so it names the next step without
// exposing an internal plugin name or an English transport message.
const (
	KeyStoreSettingsWriteNotStored = "store.settings.write_not_stored"
)

func init() {
	register(KeyStoreSettingsWriteNotStored,
		"没有保存成功，改动仍留在草稿里。请再试一次；如果还是失败，请导出日志后联系我们。",
		"The change was not saved; your edits are kept as a draft. Try again, and if it keeps failing, export the logs and contact us.")
}
