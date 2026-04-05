//go:build !windows

package metrics

import "syscall"

func collectDisk() (used int64, total int64, err error) {
	var stat syscall.Statfs_t
	err = syscall.Statfs("/", &stat)
	if err != nil {
		return 0, 0, err
	}

	total = int64(stat.Blocks) * int64(stat.Bsize)
	free := int64(stat.Bfree) * int64(stat.Bsize)
	used = total - free
	return used, total, nil
}

func collectProcesses() ([]ProcessInfo, error) {
	return nil, nil
}
