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

// contains is a simple function that checks for the presence of a string value
// within a slice of strings
func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}

	return false
}
