package main

import "errors"

// phoneDeviceAction uses only the bridge-owned original remote runtime.
// No legacy desktop credential cache is edited or used to identify a device.
func (s *mainState) phoneDeviceAction(operation, id, name string) error {
	s.mu.Lock()
	quitting, win := s.quitting, s.win
	s.mu.Unlock()
	if quitting || s.cfgStore == nil || !s.cfgStore.Get().RemoteEnabled {
		return errors.New("enable and refresh the phone runtime before managing devices")
	}
	manager := s.phoneRuntimeManager()
	if err := manager.DeviceAction(operation, id, name); err != nil {
		return err
	}
	go func() {
		manager.WaitDeviceAction()
		s.mu.Lock()
		closed := s.quitting
		s.mu.Unlock()
		if !closed && win != nil {
			win.Update(func() {})
		}
	}()
	return nil
}
