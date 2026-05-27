package cluster

import (
	"path/filepath"
	"runtime"
)

// GetServiceRootPath detects the full path to the service root folder
// which corresponds a service name by checking currently running file and going up along the folder tree until the target folder reached
func GetServiceRootPath(serviceName string) string {
	_, b, _, _ := runtime.Caller(0)
	var base string
	path := filepath.Dir(b)
	for i := 0; ; i++ {
		// ensure from hanging on some unknown file structure
		if i == 50 {
			return ""
		}
		base = filepath.Base(path)
		if base == serviceName {
			abs, _ := filepath.Abs(path)
			return abs
		}
		if base == "/" {
			return ""
		}
		path = filepath.Join(path, "..")
	}
}
