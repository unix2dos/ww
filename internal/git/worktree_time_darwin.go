//go:build darwin

package git

import (
	"os"
	"syscall"
)

func birthTimeUnixNano(info os.FileInfo) (int64, bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	if stat.Birthtimespec.Sec == 0 && stat.Birthtimespec.Nsec == 0 {
		return 0, false
	}
	return stat.Birthtimespec.Sec*1e9 + stat.Birthtimespec.Nsec, true
}
