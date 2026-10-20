package remote

import (
	"context"
	"fmt"
	"strings"
	"whaleisle/internal/dsh"
	"whaleisle/internal/store/storelocale"
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
					v.Error = storelocale.T(storelocale.KeyStoreRemoteRemovalCouldNotBeVerifiedTheHostS)
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
		v.Notice = fmt.Sprintf(storelocale.T(storelocale.KeyStoreRemoteDeletedMachine), name)
		if refreshErr != nil {
			v.Notice += storelocale.T(storelocale.KeyStoreRemoteStatusRefreshFailedRefreshToCheckT)
		}
	})
}
