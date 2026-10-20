package i18n

// Register adds one key to both catalogs.
//
// Feature packages keep their own catalog file (for example
// catalog_chat.go) that declares the key constant and calls Register from
// init, so parallel feature work never edits the shared map literals in
// i18n.go. Registration is init-only: Register must not be called after the
// application starts serving UI, because Lookup reads the maps without a
// write lock.
//
// An existing key keeps its current text; Register never overwrites a catalog
// entry. Empty keys are ignored.
func Register(key, zh, enText string) {
	if key == "" {
		return
	}
	if _, exists := zhCN[key]; !exists {
		zhCN[key] = zh
	}
	if _, exists := en[key]; !exists {
		en[key] = enText
	}
}

// Registered reports whether key exists in both catalogs.
func Registered(key string) bool {
	if key == "" {
		return false
	}
	_, zh := zhCN[key]
	_, enCatalog := en[key]
	return zh && enCatalog
}
