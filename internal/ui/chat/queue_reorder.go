package chat

import (
	"github.com/egoist/mygo/ui"
	"slices"
	"whaleisle/internal/store/session"
)

type queueDrag struct{ Scope, ID string }

func queueDropIndex(rows []session.PlanQueueItem, source, target string, after bool) (int, bool) {
	from := slices.IndexFunc(rows, func(row session.PlanQueueItem) bool { return row.ID == source })
	to := slices.IndexFunc(rows, func(row session.PlanQueueItem) bool { return row.ID == target })
	if from < 0 || to < 0 || from == to {
		return 0, false
	}
	if after {
		to++
	}
	if from < to {
		to--
	}
	return to, from != to
}

func queueDropTarget(row ui.Element, m PlanBoardModel, target string, accent ui.Color, move func(string, int)) {
	accept := func(drag queueDrag, after bool) (int, bool) {
		if drag.Scope != m.Board.QueueActions.Scope || !m.Board.QueueActions.Can(session.QueueActionMove, session.PlanQueueItem{ID: drag.ID}) || m.QueueEditor.ItemID != "" {
			return 0, false
		}
		return queueDropIndex(m.Board.Queue, drag.ID, target, after)
	}
	if drag, over := ui.DragOver[queueDrag](row); over {
		_, y, _ := row.PointerPosition()
		after := y >= row.Bounds().H/2
		if _, ok := accept(drag, after); ok {
			row.DrawOver(func(p *ui.Painter, r ui.Rect) {
				y := r.Y
				if after {
					y += r.H - 2
				}
				p.Fill(ui.Rect{X: r.X, Y: y, W: r.W, H: 2}, accent, 1)
			})
		}
	}
	if drag, _, y, dropped := ui.DropAt[queueDrag](row); dropped {
		r := row.Bounds()
		if to, ok := accept(drag, y >= r.Y+r.H/2); ok {
			move(drag.ID, to)
		}
	}
}
