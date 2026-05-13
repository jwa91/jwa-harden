// Package version exposes build-time constants set via -X ldflags.
package version

import "fmt"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func String() string {
	return fmt.Sprintf("jwa-harden %s (commit %s, built %s)", version, commit, date)
}
