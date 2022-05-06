package httpmock_test

import (
	"net/http"
	"testing"

	. "github.com/jarcoal/httpmock"
)

func TestActivateNonDefaultRace(t *testing.T) {
	for i := 0; i < 10; i++ {
		go ActivateNonDefault(&http.Client{})
	}
}
