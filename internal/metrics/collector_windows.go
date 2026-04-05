//go:build windows

package metrics

import (
	"syscall"
	"unsafe"
)

func collectDisk() (used int64, total int64, err error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceEx := kernel32.NewProc("GetDiskFreeSpaceExW")

	root, rootErr := syscall.UTF16PtrFromString("C:\\")
	if rootErr != nil {
		return 0, 0, rootErr
	}

	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes int64

	ret, _, callErr := getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(root)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalNumberOfBytes)),
		uintptr(unsafe.Pointer(&totalNumberOfFreeBytes)),
	)

	if ret == 0 {
		return 0, 0, callErr
	}

	return totalNumberOfBytes - totalNumberOfFreeBytes, totalNumberOfBytes, nil
}

func collectProcesses() ([]ProcessInfo, error) {
	return nil, nil
}
