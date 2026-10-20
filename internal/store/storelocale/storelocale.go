// Package storelocale localizes copy that originates below the UI layer: store
// error and status text plus desktop runtime messages that end up in banners,
// rows and toasts.
//
// The active locale already lives in internal/ui/i18n (set from config at start
// up and when the user switches language). This package only registers its keys
// into that catalog and resolves them through it, so store and desktop copy
// cannot drift from the UI language and there is no second locale state to
// synchronise. It deliberately keeps no catalog of its own.
package storelocale

import ui18n "whaleisle/internal/ui/i18n"

// T resolves key through the shared catalog (zh-CN, then English, then key).
func T(key string) string { return ui18n.T(key) }

// Locale reports the active catalog id, for the rare message that has to branch
// on the language instead of using a full sentence per locale.
func Locale() string { return ui18n.Locale() }

// register adds one key to the shared catalog. Generated key files call it from
// init; registration is init-only for the same reason i18n.Register is.
func register(key, zh, en string) { ui18n.Register(key, zh, en) }
