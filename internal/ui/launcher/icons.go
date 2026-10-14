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
		switch id {
		case TabComponents:
			// Electron: wireframe cube (stroke), not solid hexagon fill.
			path := iconPath(id, r.X, r.Y)
			if path != nil {
				p.StrokePath(path, 1.25, fg)
			}
			// Mid seam of the isometric cube.
			seam := &ui.Path{}
			seam.MoveTo(r.X+8, r.Y+1.5)
			seam.LineTo(r.X+8, r.Y+8)
			seam.LineTo(r.X+2.8, r.Y+11.6)
			p.StrokePath(seam, 1.1, fg)
			seam2 := &ui.Path{}
			seam2.MoveTo(r.X+8, r.Y+8)
			seam2.LineTo(r.X+13.2, r.Y+11.6)
			p.StrokePath(seam2, 1.1, fg)
			return
		case TabVersions:
			// launcher.html versions: list-in-rectangle SVG path.
			path := iconPath(TabVersions, r.X, r.Y)
			if path != nil {
				p.FillPath(path, fg)
			}
			// Outer frame from launcher.html path (list lines alone omit the rect).
			frame := &ui.Path{}
			frame.MoveTo(r.X+1.4, r.Y+1.5)
			frame.LineTo(r.X+14.6, r.Y+1.5)
			frame.LineTo(r.X+14.6, r.Y+14.5)
			frame.LineTo(r.X+1.4, r.Y+14.5)
			frame.Close()
			p.StrokePath(frame, 1.1, fg)
			return
		case TabPlugins:
			// launcher.html plugins: puzzle-piece fill (not settings gear).
			path := iconPath(TabPlugins, r.X, r.Y)
			if path != nil {
				p.FillPath(path, fg)
			}
			return
		}
		path := iconPath(id, r.X, r.Y)
		if path == nil {
			return
		}
		p.FillPath(path, fg)
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

func iconPath(id string, dx, dy float32) *ui.Path {
	p := &ui.Path{}
	m := func(x, y float32) { p.MoveTo(dx+x, dy+y) }
	l := func(x, y float32) { p.LineTo(dx+x, dy+y) }
	switch id {
	case TabHome:
		m(8.06, 1.72)
		l(1.7, 5.4)
		l(1.3, 6.2)
		l(1.3, 13.6)
		l(5.98, 13.6)
		l(5.98, 9.6)
		l(9.82, 9.6)
		l(9.82, 13.6)
		l(14.7, 13.6)
		l(14.7, 6.2)
		p.Close()
	case TabComponents:
		m(8, 1.5)
		l(2.8, 4.4)
		l(2.8, 11.6)
		l(8, 14.5)
		l(13.2, 11.6)
		l(13.2, 4.4)
		p.Close()
	case TabVersions:
		m(3.1, 3.2)
		l(12.9, 3.2)
		l(12.9, 4.6)
		l(3.1, 4.6)
		p.Close()
		m(3.1, 7.3)
		l(12.9, 7.3)
		l(12.9, 8.7)
		l(3.1, 8.7)
		p.Close()
		m(3.1, 11.4)
		l(12.9, 11.4)
		l(12.9, 12.8)
		l(3.1, 12.8)
		p.Close()
	case TabImport:
		m(8.72, 1.3)
		l(8.72, 9)
		l(11.9, 6.26)
		l(12.92, 7.28)
		l(8.46, 11.5)
		l(7.54, 11.5)
		l(3.08, 7.28)
		l(4.1, 6.26)
		l(7.28, 9)
		l(7.28, 1.3)
		p.Close()
	case TabPlugins:
		m(6.2, 1.4)
		l(9.8, 1.4)
		l(9.8, 3.6)
		l(12.4, 3.6)
		l(13.5, 4.7)
		l(13.5, 7.3)
		l(16, 7.3)
		l(16, 10.9)
		l(13.5, 10.9)
		l(13.5, 13.5)
		l(12.4, 14.6)
		l(9.8, 14.6)
		l(9.8, 16)
		l(6.2, 16)
		l(6.2, 13.5)
		l(3.6, 13.5)
		l(2.5, 12.4)
		l(2.5, 9.8)
		l(0, 9.8)
		l(0, 6.2)
		l(2.5, 6.2)
		l(2.5, 3.6)
		l(3.6, 2.5)
		l(6.2, 2.5)
		p.Close()
	case TabSettings:
		// Coarse gear matching Electron launcher.html settings path silhouette.
		m(8, 1.4)
		l(9.1, 1.4)
		l(9.5, 3.2)
		l(11.1, 3.8)
		l(12.6, 2.8)
		l(13.5, 3.7)
		l(12.5, 5.2)
		l(13.1, 6.8)
		l(14.9, 7.2)
		l(14.9, 8.8)
		l(13.1, 9.2)
		l(12.5, 10.8)
		l(13.5, 12.3)
		l(12.6, 13.2)
		l(11.1, 12.2)
		l(9.5, 12.8)
		l(9.1, 14.6)
		l(6.9, 14.6)
		l(6.5, 12.8)
		l(4.9, 12.2)
		l(3.4, 13.2)
		l(2.5, 12.3)
		l(3.5, 10.8)
		l(2.9, 9.2)
		l(1.1, 8.8)
		l(1.1, 7.2)
		l(2.9, 6.8)
		l(3.5, 5.2)
		l(2.5, 3.7)
		l(3.4, 2.8)
		l(4.9, 3.8)
		l(6.5, 3.2)
		l(6.9, 1.4)
		p.Close()
		p.Circle(dx+8, dy+8, 2.4)
	default:
		return nil
	}
	return p
}
