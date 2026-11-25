//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procGetVolumePathNameW = kernel32.NewProc("GetVolumePathNameW")
	procGetVolumeInfoW     = kernel32.NewProc("GetVolumeInformationW")
	procGetDriveTypeW      = kernel32.NewProc("GetDriveTypeW")
)

const driveRemote = 4 // DRIVE_REMOTE

func platformLookupVolume(path string) (volumeInfo, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return volumeInfo{}, err
	}

	rootBuf := make([]uint16, 1024)
	r, _, e := procGetVolumePathNameW.Call(
		uintptr(unsafe.Pointer(p)),
		uintptr(unsafe.Pointer(&rootBuf[0])),
		uintptr(len(rootBuf)))
	if r == 0 {
		return volumeInfo{}, fmt.Errorf("GetVolumePathName: %w", e)
	}
	root := syscall.UTF16ToString(rootBuf)
	rootPtr, err := syscall.UTF16PtrFromString(root)
	if err != nil {
		return volumeInfo{}, err
	}

	labelBuf := make([]uint16, 261)
	r, _, e = procGetVolumeInfoW.Call(
		uintptr(unsafe.Pointer(rootPtr)),
		uintptr(unsafe.Pointer(&labelBuf[0])),
		uintptr(len(labelBuf)),
		0, 0, 0, 0, 0)
	if r == 0 {
		return volumeInfo{}, fmt.Errorf("GetVolumeInformation(%s): %w", root, e)
	}

	dt, _, _ := procGetDriveTypeW.Call(uintptr(unsafe.Pointer(rootPtr)))

	return volumeInfo{
		Root:   root,
		Label:  syscall.UTF16ToString(labelBuf),
		Remote: dt == driveRemote,
	}, nil
}
