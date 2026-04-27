package version

import "fmt"

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func String() string {
	return fmt.Sprintf("whoiscached %s (commit %s, built %s)", Version, Commit, BuildTime)
}
