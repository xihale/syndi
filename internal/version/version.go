// Package version holds build metadata injected at link time:
//
//	go build -ldflags "-X github.com/xihale/syndi/internal/version.Version=v0.1.0"
//
// Unset values mean a plain source build (go run / go build without ldflags).
package version

import (
	"fmt"
	"runtime"
)

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// Summary returns a one-line build description for logs and the version command.
func Summary() string {
	return fmt.Sprintf("syndi %s (commit %s, built %s, go %s, %s/%s)",
		Version, Commit, Date, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}
