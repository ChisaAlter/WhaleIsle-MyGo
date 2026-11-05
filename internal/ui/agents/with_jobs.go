package agents

import "github.com/egoist/mygo/ui"

// The Jobs summary is part of the same scrolling directory body, after agents.
func ViewWithJobs(c *ui.Context, m Model, jobs func(*ui.Context)) {
	if m.Store != nil {
		navigationView(c, m, jobs)
		return
	}
	View(c, m)
}
