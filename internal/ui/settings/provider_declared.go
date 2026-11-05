package settings

func configurationDeclaredProvider(m Model, namespace, route string) bool {
	if m.Store == nil {
		return false
	}
	rows, _, _ := m.Store.ConfigurableProviders()
	for _, row := range rows {
		if row.SettingsNs == namespace && row.Provider == route {
			return row.Declared != nil && *row.Declared
		}
	}
	return false
}
