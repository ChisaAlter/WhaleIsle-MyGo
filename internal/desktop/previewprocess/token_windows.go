//go:build windows && (amd64 || arm64)

package previewprocess

import (
	"encoding/binary"
	"errors"
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

// TOKEN_INFORMATION_CLASS values from winnt.h. x/sys currently stops its
// named constants before these Windows 8 AppContainer additions.
const (
	tokenIsAppContainer = 29
	tokenCapabilities = 30
	tokenAppContainerSID = 31
)

func tokenData(token windows.Token, class uint32) ([]byte, error) {
	var n uint32
	err := windows.GetTokenInformation(token, class, nil, 0, &n)
	if err != windows.ERROR_INSUFFICIENT_BUFFER || n < 4 || n > 64*1024 {
		return nil, fmt.Errorf("cannot size token information %d: %v", class, err)
	}
	b := make([]byte, n)
	if err = windows.GetTokenInformation(token, class, &b[0], n, &n); err != nil { return nil, err }
	if n > uint32(len(b)) || n < 4 { return nil, errors.New("invalid token information length") }
	return b[:n], nil
}

func verifyToken(token windows.Token, expectedSID string) error {
	flag, err := tokenData(token, tokenIsAppContainer)
	if err != nil { return err }
	if binary.LittleEndian.Uint32(flag) != 1 { return errors.New("process is not an AppContainer") }
	caps, err := tokenData(token, tokenCapabilities)
	if err != nil { return err }
	// Reject every capability, not merely the known network capability names.
	if binary.LittleEndian.Uint32(caps) != 0 { return errors.New("preview process has capabilities") }
	info, err := tokenData(token, tokenAppContainerSID)
	if err != nil { return err }
	if len(info) < int(unsafe.Sizeof(uintptr(0))) { return errors.New("missing AppContainer SID") }
	sid := *(**windows.SID)(unsafe.Pointer(&info[0]))
	if sid == nil { return errors.New("missing AppContainer SID") }
	got := sid.String()
	runtime.KeepAlive(info)
	if got == "" || got != expectedSID { return errors.New("unexpected AppContainer SID") }
	return nil
}

func verifyProcessToken(process windows.Handle, expectedSID string) error {
	var token windows.Token
	if err := windows.OpenProcessToken(process, windows.TOKEN_QUERY, &token); err != nil { return err }
	defer token.Close()
	return verifyToken(token, expectedSID)
}

func verifyCurrentToken(expectedSID string) error {
	return verifyProcessToken(windows.CurrentProcess(), expectedSID)
}

func jobPIDs(job windows.Handle) ([]uint32, error) {
	// The job has a hard active-process limit of 32. Keep generous room and
	// reject truncation; never mistake a partial enumeration for an attestation.
	var data struct {
		Assigned uint32
		Listed uint32
		PIDs [128]uintptr
	}
	var returned uint32
	err := windows.QueryInformationJobObject(job, windows.JobObjectBasicProcessIdList, uintptr(unsafe.Pointer(&data)), uint32(unsafe.Sizeof(data)), &returned)
	if err != nil { return nil, err }
	if data.Listed > uint32(len(data.PIDs)) || data.Assigned != data.Listed { return nil, errors.New("incomplete preview job enumeration") }
	pids := make([]uint32, 0, data.Listed)
	for _, p := range data.PIDs[:data.Listed] {
		if p == 0 || p > 0xffffffff { return nil, errors.New("invalid preview process ID") }
		pids = append(pids, uint32(p))
	}
	return pids, nil
}

var isProcessInJobProc = windows.NewLazySystemDLL("kernel32.dll").NewProc("IsProcessInJob")

func verifyInJob(process, job windows.Handle) error {
	var inJob int32
	ok, _, err := isProcessInJobProc.Call(uintptr(process), uintptr(job), uintptr(unsafe.Pointer(&inJob)))
	if ok == 0 { return fmt.Errorf("IsProcessInJob: %w", err) }
	if inJob != 1 { return errors.New("process is outside preview job") }
	return nil
}

func verifyTree(job windows.Handle, sid string, helperPID, browserPID uint32) error {
	if helperPID == 0 || browserPID == 0 || helperPID == browserPID { return errors.New("invalid preview browser identity") }
	pids, err := jobPIDs(job)
	if err != nil { return err }
	foundHelper, foundBrowser := false, false
	for _, pid := range pids {
		process, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
		if err != nil { return fmt.Errorf("query preview process %d: %w", pid, err) }
		err = verifyInJob(process, job)
		if err == nil { err = verifyProcessToken(process, sid) }
		windows.CloseHandle(process)
		if err != nil { return fmt.Errorf("preview process %d: %w", pid, err) }
		foundHelper = foundHelper || pid == helperPID
		foundBrowser = foundBrowser || pid == browserPID
	}
	if !foundHelper || !foundBrowser { return errors.New("preview job lacks the expected helper/browser") }
	return nil
}
