package httpmock

import (
	"net/http"
	"strings"

	"github.com/goware/urlx"
)

// StubbedRequest is used to capture data about a new stubbed request. It wraps
// up the Method and URL along with optional http.Header struct, and also holds
// the Responder.
type StubbedRequest struct {
	Method    string
	URL       string
	Header    *http.Header
	Responder *Responder
}

// normalizedKey is a helper function that returns a normalized key for a
// method/url pair.
func normalizedKey(method, url string) (string, error) {
	u, err := urlx.Parse(url)
	if err != nil {
		return "", err
	}

	normalized, err := urlx.Normalize(u)
	if err != nil {
		return "", err
	}

	return strings.ToUpper(method) + " " + normalized, nil
}
