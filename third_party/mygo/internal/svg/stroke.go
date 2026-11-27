package svg

import "math"

// Line caps and joins.
const (
	capButt uint8 = iota
	capRound
	capSquare
)

const (
	joinMiter uint8 = iota
	joinRound
	joinBevel
)

// stroker outlines polylines as continuous offset contours instead of per-segment
// quads and corner wedges: each edge of a stroke is emitted once so that the
// non-zero rasterizer never double-accumulates coverage along flattened curves.
// All stroke outlines wind counter-clockwise in screen coordinates (negative
// signed area, winding -1 on the stroke; closed holes wind +1 to cancel to 0),
// so crossing subpaths never cancel each other out.
type stroker struct {
	hw        float64 // half the width
	join, cap uint8
	miter     float64 // the miter limit
	tol       float64 // how far arcs may stray from circles
	out       *polys
	pts       []point
}

func (s *stroker) stroke(in *polys) {
	for i := range in.len() {
		closed := in.closed[i]
		pts := s.pts[:0]
		for _, p := range in.poly(i) {
			if len(pts) == 0 || math.Hypot(p.x-pts[len(pts)-1].x, p.y-pts[len(pts)-1].y) > 0 {
				pts = append(pts, p)
			}
		}
		if closed && len(pts) > 1 && math.Hypot(pts[0].x-pts[len(pts)-1].x, pts[0].y-pts[len(pts)-1].y) == 0 {
			pts = pts[:len(pts)-1]
		}
		s.pts = pts
		n := len(pts)
		switch {
		case n == 0:
			continue
		case n == 1:
			s.dot(pts[0])
		case closed && n == 2:
			// A closed 2-point subpath folds back on itself along one segment:
			// a single offsetLoop traces both sides and both 180° joins once.
			s.offsetLoop(pts)
		case closed:
			// Each side of a closed polyline is its own loop; reversing pts
			// traces the opposite side the other way so the enclosed hole
			// cancels to winding 0 while the stroke ring keeps winding -1.
			s.offsetLoop(pts)
			for k := range n / 2 {
				pts[k], pts[n-1-k] = pts[n-1-k], pts[k]
			}
			s.offsetLoop(pts)
		default:
			s.offsetOpen(pts)
		}
	}
}

// full reports whether the outline has all the points it may have.
func (s *stroker) full() bool { return len(s.out.pts) >= 2*maxPoints }

func unit(a, b point) point {
	dx, dy := b.x-a.x, b.y-a.y
	l := math.Hypot(dx, dy)
	if l == 0 || math.IsNaN(l) || math.IsInf(l, 0) {
		return point{1, 0}
	}
	return point{dx / l, dy / l}
}

func (s *stroker) edge(a, b point) (d, n point) {
	d = unit(a, b)
	return d, point{-d.y * s.hw, d.x * s.hw}
}

// offsetLoop traces one offset contour of a closed polyline hw units to the
// left of each directed edge (normal (-dy, dx)), turning around outer corners
// with the configured join style and routing inner corners through the vertex.
func (s *stroker) offsetLoop(pts []point) {
	if s.full() {
		return
	}
	n := len(pts)
	o := s.out
	d0, n0 := s.edge(pts[0], pts[1%n])
	o.start(point{pts[0].x + n0.x, pts[0].y + n0.y})
	for k := range n {
		b := pts[(k+1)%n]
		o.add(point{b.x + n0.x, b.y + n0.y})
		d1, n1 := s.edge(b, pts[(k+2)%n])
		s.joinCorner(b, d0, n0, d1, n1)
		d0, n0 = d1, n1
	}
	o.end(true)
}

// offsetOpen outlines an open polyline as a single closed CCW contour: forward
// along one offset side, around the end cap, back along the opposite offset
// side, and around the start cap.
func (s *stroker) offsetOpen(pts []point) {
	if s.full() {
		return
	}
	n := len(pts)
	o := s.out
	d0, n0 := s.edge(pts[0], pts[1])
	o.start(point{pts[0].x + n0.x, pts[0].y + n0.y})
	for k := 0; k < n-1; k++ {
		b := pts[k+1]
		o.add(point{b.x + n0.x, b.y + n0.y})
		if k+1 < n-1 {
			d1, n1 := s.edge(pts[k+1], pts[k+2])
			s.joinCorner(b, d0, n0, d1, n1)
			d0, n0 = d1, n1
		}
	}
	s.capEnd(pts[n-1], d0, n0)
	d0, n0 = point{-d0.x, -d0.y}, point{-n0.x, -n0.y}
	for k := n - 2; k >= 0; k-- {
		b := pts[k]
		o.add(point{b.x + n0.x, b.y + n0.y})
		if k > 0 {
			d1, n1 := s.edge(pts[k], pts[k-1])
			s.joinCorner(b, d0, n0, d1, n1)
			d0, n0 = d1, n1
		}
	}
	s.capEnd(pts[0], d0, n0)
	o.end(true)
}

// joinCorner transitions the active contour at vertex b from offset point b+n0
// (direction d0) to b+n1 (direction d1).
func (s *stroker) joinCorner(b, d0, n0, d1, n1 point) {
	o := s.out
	cross := d0.x*d1.y - d0.y*d1.x
	dot := d0.x*d1.x + d0.y*d1.y
	if math.Abs(cross) < 1e-12 && dot > 0 {
		return // straight on
	}
	end := point{b.x + n1.x, b.y + n1.y}
	if cross > 0 {
		// Inside of the turn: route through the vertex where the stroke covers itself.
		o.add(b)
		o.add(end)
		return
	}
	// Outside of the turn (cross < 0) or 180° foldback (cross == 0, dot < 0).
	switch s.join {
	case joinRound:
		turn := math.Atan2(cross, dot)
		if cross == 0 {
			turn = -math.Pi
		}
		s.arcTo(b, n0, end, turn)
	case joinBevel:
		o.add(end)
	default:
		// The miter's length over the stroke's width is 1/cos(turn/2).
		cosHalf := math.Sqrt(math.Max((1+dot)/2, 0))
		if cross < 0 && cosHalf > 1e-9 && cosHalf*s.miter >= 1 {
			u := unit(point{}, point{n0.x + n1.x, n0.y + n1.y})
			l := s.hw / cosHalf
			o.add(point{b.x + u.x*l, b.y + u.y*l})
		}
		o.add(end)
	}
}

// capEnd transitions the active open contour at endpoint p (arriving along d
// at p+n) across the cap to p-n.
func (s *stroker) capEnd(p, d, n point) {
	o := s.out
	end := point{p.x - n.x, p.y - n.y}
	switch s.cap {
	case capRound:
		s.arcTo(p, n, end, -math.Pi)
	case capSquare:
		ex, ey := d.x*s.hw, d.y*s.hw
		o.add(point{p.x + n.x + ex, p.y + n.y + ey})
		o.add(point{p.x - n.x + ex, p.y - n.y + ey})
		o.add(end)
	default: // capButt
		o.add(end)
	}
}

// dot draws a subpath of no length: a disc with round caps, a square with
// square caps, both wound CCW to match open and closed stroke contours.
func (s *stroker) dot(p point) {
	if s.full() {
		return
	}
	o := s.out
	switch s.cap {
	case capRound:
		steps := max(s.arcSteps(2*math.Pi), 8)
		o.start(point{p.x + s.hw, p.y})
		for k := 1; k < steps; k++ {
			sin, cos := math.Sincos(-2 * math.Pi * float64(k) / float64(steps))
			o.add(point{p.x + cos*s.hw, p.y + sin*s.hw})
		}
		o.end(true)
	case capSquare:
		h := s.hw
		o.start(point{p.x - h, p.y - h})
		o.add(point{p.x - h, p.y + h})
		o.add(point{p.x + h, p.y + h})
		o.add(point{p.x + h, p.y - h})
		o.end(true)
	}
}

func (s *stroker) arcSteps(sweep float64) int {
	// Keep chord sagitta near 0.02px (matching ui/path.go) so round caps and
	// joins stay accurate without inflating polygonal radius.
	tol := s.tol * 0.2
	if tol <= 0 || s.hw <= 0 {
		return 1
	}
	step := 2 * math.Acos(math.Max(1-tol/s.hw, -1))
	if !(step > 0) {
		return 1
	}
	return max(1, min(int(math.Ceil(math.Abs(sweep)/step)), 256))
}

// arcTo appends points along the circle of radius hw centered at c from c+from
// sweeping sweep radians, terminating at end.
func (s *stroker) arcTo(c, from, end point, sweep float64) {
	o := s.out
	steps := s.arcSteps(sweep)
	if steps > 1 && !s.full() {
		a0 := math.Atan2(from.y, from.x)
		for k := 1; k < steps; k++ {
			sin, cos := math.Sincos(a0 + sweep*float64(k)/float64(steps))
			o.add(point{c.x + cos*s.hw, c.y + sin*s.hw})
		}
	}
	o.add(end)
}

// dash cuts the polylines of in into the dashes of a dash array (of even
// length), starting offset into it, as open polylines in out. limit
// bounds the number of dashes.
func dash(in *polys, dashes []float64, offset float64, out *polys, limit int) {
	var total float64
	for _, d := range dashes {
		total += d
	}
	offset = math.Mod(offset, total)
	if offset < 0 {
		offset += total
	}
	for i := range in.len() {
		pts := in.poly(i)
		// Every subpath starts the pattern anew, offset into it: in dash
		// or gap k, with left of it to go.
		k, o := 0, offset
		for o > dashes[k] {
			o -= dashes[k]
			k = (k + 1) % len(dashes)
		}
		left := dashes[k] - o
		on := k%2 == 0
		if on {
			out.start(pts[0])
		}
		n := len(pts)
		segs := n - 1
		if in.closed[i] {
			segs = n
		}
		for sg := range segs {
			a, b := pts[sg], pts[(sg+1)%n]
			l := math.Hypot(b.x-a.x, b.y-a.y)
			pos := 0.0
			for l-pos > left {
				pos += left
				t := pos / l
				p := point{a.x + (b.x-a.x)*t, a.y + (b.y-a.y)*t}
				if on {
					out.add(p)
					out.end(false)
					limit--
					if limit <= 0 {
						return
					}
				} else {
					out.start(p)
				}
				on = !on
				k = (k + 1) % len(dashes)
				left = dashes[k]
			}
			left -= l - pos
			if on {
				out.add(b)
			}
		}
		out.end(false)
	}
}
