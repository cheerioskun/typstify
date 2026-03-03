package version

import (
	"fmt"
	"runtime"
	"strconv"
	"time"
)

var (
	BinVersion     = "unknown"
	BuildTime      = "1706890000"
	BuildGoVersion = "unknown"
)

func VersionStr() string {
	return fmt.Sprintf("%s-%s %s %s-%s", BinVersion, ParsedBuildTime().Format(time.DateOnly), BuildGoVersion, runtime.GOOS, runtime.GOARCH)
}

func ParsedBuildTime() time.Time {
	t, err := strconv.ParseInt(BuildTime, 10, 64)
	if err != nil {
		panic(err)
	}

	return time.Unix(t, 0)
}
