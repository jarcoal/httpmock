package httpmock

import (
	"github.com/goware/urlx"
)

// normalizeURL is a helper function that returns a normalized url
func normalizeURL(url string) (string, error) {
	u, err := urlx.Parse(url)
	if err != nil {
		return "", err
	}

	normalized, err := urlx.Normalize(u)
	if err != nil {
		return "", err
	}

	return normalized, nil
}
