package launcher

import "github.com/egoist/mygo/ui"

func routeChevron(c *ui.Context, fg ui.Color, expanded bool) {
	// Exact launcher.html route-chip-chev SVG, scaled from its 16px viewBox.
	ui.Box(c).Width(12).Height(12).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		path := &ui.Path{}
		points := [][2]float32{{8, 10.6}, {3.7, 6.3}, {4.8, 5.2}, {8, 8.4}, {11.2, 5.2}, {12.3, 6.3}}
		for i, point := range points {
			x, y := point[0], point[1]
			if expanded {
				x, y = 16-x, 16-y
			}
			x, y = r.X+x*r.W/16, r.Y+y*r.H/16
			if i == 0 {
				path.MoveTo(x, y)
			} else {
				path.LineTo(x, y)
			}
		}
		path.Close()
		p.FillPath(path, fg)
	})
}

func tabIcon(c *ui.Context, id string, fg ui.Color) {
	ui.Box(c).Width(16).Height(16).Draw(func(p *ui.Painter, r ui.Rect) {
		// The six nav glyphs are the launcher.html paths, drawn as fills with the
		// box scaled from the 16-unit viewBox (navIconPaths in svgpath.go).
		d, ok := navIconPaths[id]
		if !ok {
			return
		}
		// The SVG viewport clips at the icon box (the settings path carries a few
		// coordinates past 16), so the fill is clipped the same way.
		box := min(r.W, r.H)
		p.Clip(ui.Rect{X: r.X, Y: r.Y, W: box, H: box}, 0, func() {
			p.FillPath(svgIcon(d, r.X, r.Y, box), fg)
		})
	})
}

func globeIcon(c *ui.Context, fg ui.Color) {
	// Electron rail-route globe (12×12), simplified fill matching launcher.html path.
	ui.Box(c).Width(12).Height(12).Draw(func(p *ui.Painter, r ui.Rect) {
		dx, dy := r.X, r.Y
		s := float32(12) / 16
		path := &ui.Path{}
		path.Circle(dx+8*s, dy+8*s, 7*s)
		p.StrokePath(path, 1.1, fg)
		// latitude
		lat := &ui.Path{}
		lat.MoveTo(dx+1.5*s, dy+8*s)
		lat.LineTo(dx+14.5*s, dy+8*s)
		p.StrokePath(lat, 1, fg)
		// longitude arcs (approx as vertical ellipse chords)
		lon := &ui.Path{}
		lon.MoveTo(dx+8*s, dy+1.5*s)
		lon.LineTo(dx+8*s, dy+14.5*s)
		p.StrokePath(lon, 1, fg)
	})
}
