package httpmock

import (
	"os"
)

var envVarName = "GONOMOCKS"

// Disabled returns true if the GONOMOCKS environment variable is not empty
func Disabled() bool {
	return os.Getenv(envVarName) != ""
}
