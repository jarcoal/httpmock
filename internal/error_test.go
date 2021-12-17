package internal_test

import (
	"testing"

	"github.com/jarcoal/httpmock/internal"
)

func TestErrorNoResponderFoundMethodCase(t *testing.T) {
	e := internal.NewErrorNoResponderFoundWrongMethod("pipo", "BINGO")

	if e.Error() != "no responder found for method pipo, but one matches method BINGO" {
		t.Errorf("not expected error message: %s", e)
	}

	werr := e.(*internal.ErrorNoResponderFoundWrongMethod).Unwrap()
	if werr != internal.NoResponderFound {
		t.Errorf("NoResponderFound is not wrapped, but %[1]s (%[1]T)", werr)
	}
}
