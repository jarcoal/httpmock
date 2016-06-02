package httpmock

import (
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	testcases := []struct {
		input       string
		expected    string
		expectedErr bool
	}{
		{
			"http://example.com",
			"http://example.com",
			false,
		},
	}

	for _, testcase := range testcases {
		got, err := normalizeURL(testcase.input)
		if err != nil && !testcase.expectedErr {
			t.Errorf("Unexpected error, got %#v", err)
		}

		if got != testcase.expected {
			t.Errorf("Unexepcted output, expected: %s, got %s", testcase.expected, got)
		}
	}
}
