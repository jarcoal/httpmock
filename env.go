package httpmock

import (
	"os"
)

var disableEnv bool

func init() {
	disableEnv = os.Getenv("GONOMOCKS") != ""
}

func Disabled() bool {
	return disableEnv
}
