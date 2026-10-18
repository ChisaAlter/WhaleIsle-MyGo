package remote

import (
	"context"
	"fmt"
	"strings"
	"whaleisle/internal/dsh"
)

// RemoveAtGeneration submits only a confirmation belonging to this Host.
// A failed mutation keeps its target and draft available for an explicit retry.
func (s *Store) RemoveAtGeneration(id string, generation uint64) bool {
	if strings.TrimSpace(id) == "" {
		return false
	}
	name := id
	for _, machine := range s.Snapshot().Machines {
		if machine.ID == id {
			name = machine.Name
			if name == "" {
				name = machine.Host
			}
			if name == "" {
				name = id
			}
			break
		}
	}
	return s.runMutation(func(ctx context.Context, c *dsh.Client) error {
		return request(ctx, c, "POST", "machines", map[string]string{"action": "delete", "id": id}, nil)
	}, &generation, func(v *Snapshot, refreshErr error, machines []Machine, registryLoaded bool) {
		if registryLoaded {
			for _, machine := range machines {
				if machine.ID == id {
					v.Error = "删除未能确认：Host 仍返回此机器，请刷新后重试。 / Removal could not be verified: the Host still lists this machine; refresh before retrying."
					return
				}
			}
		}
		// The delete was acknowledged. Do not show the old machine as removable
		// again if the subsequent registry/status refresh failed.
		kept := make([]Machine, 0, len(v.Machines))
		for _, machine := range v.Machines {
			if machine.ID != id {
				kept = append(kept, machine)
			}
		}
		v.Machines = kept
		if v.CurrentID == id {
			v.CurrentID = ""
			v.Connected = false
			v.Status = ConnectionStatus{}
			v.Directory = DirectorySnapshot{}
			v.Tools = ToolsSnapshot{}
		}
		v.RemovedID = id
		v.Notice = fmt.Sprintf("已删除机器「%s」 / Deleted machine %q", name, name)
		if refreshErr != nil {
			v.Notice += "；状态刷新失败，请刷新。 / Status refresh failed; refresh to check the remaining state."
		}
	})
}
