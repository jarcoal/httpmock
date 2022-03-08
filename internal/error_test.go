package internal_test

import (
	"testing"

	"github.com/jarcoal/httpmock/internal"
)

func TestErrorNoResponderFoundMistake(t *testing.T) {
	e := &internal.ErrorNoResponderFoundMistake{
		Kind:      "method",
		Orig:      "pipo",
		Suggested: "BINGO",
	}

	if e.Error() != `no responder found for method "pipo", but one matches method "BINGO"` {
		t.Errorf("not expected error message: %s", e)
	}

	werr := e.Unwrap()
	if werr != internal.NoResponderFound {
		t.Errorf("NoResponderFound is not wrapped, but %[1]s (%[1]T)", werr)
	}
}
