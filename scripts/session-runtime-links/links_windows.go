package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

var kernel = syscall.NewLazyDLL("kernel32.dll")
var removeDirectory = kernel.NewProc("RemoveDirectoryW")
var finalPathName = kernel.NewProc("GetFinalPathNameByHandleW")

func runtimeReparseTag(filename string) (uint32, bool, error) {
	name, err := syscall.UTF16PtrFromString(filename)
	if err != nil {
		return 0, false, err
	}
	var data syscall.Win32FileAttributeData
	if err = syscall.GetFileAttributesEx(name, syscall.GetFileExInfoStandard, (*byte)(unsafe.Pointer(&data))); err != nil {
		return 0, false, err
	}
	if data.FileAttributes&syscall.FILE_ATTRIBUTE_REPARSE_POINT == 0 {
		return 0, false, nil
	}
	handle, err := syscall.CreateFile(name, 0, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE, nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_OPEN_REPARSE_POINT|syscall.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return 0, true, err
	}
	defer syscall.CloseHandle(handle)
	buffer := make([]byte, 16384)
	var returned uint32
	if err = syscall.DeviceIoControl(handle, 0x000900A8, nil, 0, &buffer[0], uint32(len(buffer)), &returned, nil); err != nil {
		return 0, true, err
	}
	return binary.LittleEndian.Uint32(buffer), true, nil
}

func resolveDirectoryLink(link string) (string, error) {
	name, err := syscall.UTF16PtrFromString(link)
	if err != nil {
		return "", err
	}
	handle, err := syscall.CreateFile(name, 0, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE, nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return "", err
	}
	defer syscall.CloseHandle(handle)
	buffer := make([]uint16, 32768)
	n, _, last := finalPathName.Call(uintptr(handle), uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)), 0)
	if n == 0 || n >= uintptr(len(buffer)) {
		return "", last
	}
	result := syscall.UTF16ToString(buffer[:n])
	if strings.HasPrefix(result, `\\?\UNC\`) {
		result = `\\` + strings.TrimPrefix(result, `\\?\UNC\`)
	} else {
		result = strings.TrimPrefix(result, `\\?\`)
	}
	return filepath.Clean(result), nil
}

// Construct a mount-point reparse buffer directly. Junction creation needs no
// Developer Mode, privilege, command shell, or console process.
func createDirectoryLink(target, link string) error {
	if err := os.Mkdir(link, 0700); err != nil {
		return err
	}
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(link)
		}
	}()
	printName := filepath.Clean(target)
	substitute := `\??\` + strings.TrimPrefix(printName, `\\?\`)
	if strings.HasPrefix(printName, `\\`) {
		substitute = `\??\UNC\` + strings.TrimPrefix(printName, `\\`)
	}
	sub := utf16.Encode([]rune(substitute))
	printed := utf16.Encode([]rune(printName))
	dataLen := 8 + (len(sub)+1+len(printed)+1)*2
	data := make([]byte, 8+dataLen)
	binary.LittleEndian.PutUint32(data, 0xA0000003)
	binary.LittleEndian.PutUint16(data[4:], uint16(dataLen))
	binary.LittleEndian.PutUint16(data[10:], uint16(len(sub)*2))
	binary.LittleEndian.PutUint16(data[12:], uint16((len(sub)+1)*2))
	binary.LittleEndian.PutUint16(data[14:], uint16(len(printed)*2))
	for i, c := range sub {
		binary.LittleEndian.PutUint16(data[16+i*2:], c)
	}
	for i, c := range printed {
		binary.LittleEndian.PutUint16(data[16+(len(sub)+1+i)*2:], c)
	}
	name, err := syscall.UTF16PtrFromString(link)
	if err != nil {
		return err
	}
	handle, err := syscall.CreateFile(name, syscall.GENERIC_WRITE, 0, nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_OPEN_REPARSE_POINT|syscall.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(handle)
	var returned uint32
	if err = syscall.DeviceIoControl(handle, 0x000900A4, &data[0], uint32(len(data)), nil, 0, &returned, nil); err != nil {
		return err
	}
	ok = true
	return nil
}
func removeDirectoryLink(link string) error {
	name, err := syscall.UTF16PtrFromString(link)
	if err != nil {
		return err
	}
	r, _, e := removeDirectory.Call(uintptr(unsafe.Pointer(name)))
	if r == 0 {
		return e
	}
	return nil
}
