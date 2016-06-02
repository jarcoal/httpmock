package httpmock

import (
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	testcases := []struct {
		inputs      []string
		expected    string
		expectedErr bool
	}{
		{
			[]string{
				"www.example.com",
				"www.example.com:80",
				"http://www.example.com",
				"HTTP://WWW.eXamPle.Com:80",
			},
			"http://www.example.com",
			false,
		},
		{
			[]string{
				"http://www.example.com/a/b/index.html?foo=val&bar=val#t=20",
				"www.example.com:80/a/b///../x/../../index.html?foo=val&bar=val#t=20",
			},
			"http://www.example.com/a/b/index.html?bar=val&foo=val#t=20",
			false,
		},
		{
			[]string{
				"<funnytag>",
				"javascript:evilFunc()",
				"anotherscheme:garbage",
			},
			"",
			true,
		},
	}

	for _, testcase := range testcases {
		for _, input := range testcase.inputs {
			got, err := normalizeURL(input)
			if testcase.expectedErr {
				if err == nil {
					t.Errorf("Expected error for '%s', got none", input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error, got %#v", err)
				}

				if got != testcase.expected {
					t.Errorf("Unexepcted output, expected: %s, got %s", testcase.expected, got)
				}
			}
		}
	}
}
