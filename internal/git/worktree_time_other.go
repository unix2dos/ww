//go:build !darwin

package git

import "os"

func birthTimeUnixNano(info os.FileInfo) (int64, bool) {
	return 0, false
}
