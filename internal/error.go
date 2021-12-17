package internal

import (
	"errors"
	"fmt"
	"strings"
)

// NoResponderFound is returned when no responders are found for a
// given HTTP method and URL.
var NoResponderFound = errors.New("no responder found") // nolint: revive

// errorNoResponderFoundMethodCase encapsulates a NoResponderFound
// error probably due to the method not upper-cased.
type ErrorNoResponderFoundMethodCase string

// Unwrap implements the interface needed by errors.Unwrap.
func (e ErrorNoResponderFoundMethodCase) Unwrap() error {
	return NoResponderFound
}

// Error implements error interface.
func (e ErrorNoResponderFoundMethodCase) Error() string {
	return fmt.Sprintf("%s for method %s, but one matches method %s",
		NoResponderFound,
		string(e),
		strings.ToUpper(string(e)),
	)
}
