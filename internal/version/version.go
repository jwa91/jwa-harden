// Package version exposes build-time constants set via -X ldflags.
package version

import "fmt"

var (
	Current = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func String() string {
	return fmt.Sprintf("jwa-harden %s (commit %s, built %s)", Current, Commit, Date)
}
