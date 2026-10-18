package wallpapers

// ApplyCursorPreset preserves the original e9 palette/speed/size bundles.
func (p *Preferences) ApplyCursorPreset(id string) {
	p.CursorPreset = id
	p.CursorColors = [6]string{}
	p.CursorSpeed = 100
	p.CursorSize = 100
	switch id {
	case "rainbow":
		p.CursorColors = [6]string{"#f87171", "#fbbf24", "#4ade80", "#38bdf8", "#a78bfa", "#f472b6"}
		p.CursorSpeed = 120
	case "aurora":
		p.CursorColors = [6]string{"#34d399", "#22d3ee", "#a78bfa", "#4ade80", "#38bdf8", "#818cf8"}
		p.CursorSize = 110
	case "sunset":
		p.CursorColors = [6]string{"#fb923c", "#fb7185", "#f472b6", "#fbbf24", "#f87171", "#fdba74"}
		p.CursorSpeed = 90
		p.CursorSize = 120
	case "ocean":
		p.CursorColors = [6]string{"#38bdf8", "#2dd4bf", "#60a5fa", "#818cf8", "#22d3ee", "#67e8f9"}
	case "sakura":
		p.CursorColors = [6]string{"#f9a8d4", "#f472b6", "#c4b5fd", "#fda4af", "#fbcfe8", "#e879f9"}
		p.CursorSpeed = 80
		p.CursorSize = 90
	}
}
