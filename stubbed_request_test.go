package httpmock

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

func TestMatches(t *testing.T) {
	testcases := []struct {
		method        string
		requestMethod string
		url           string
		requestURL    string
		expectedErr   bool
	}{
		{
			"get",
			"GET",
			"http://example.com",
			"http://example.com",
			false,
		},
		{
			"get",
			"GET",
			"ExAmPlE.com?foo=val&bar=val#t=boo",
			"http://example.com?bar=val&foo=val#t=boo",
			false,
		},
		{
			"Get",
			"GET",
			"http://example.com:80/?bar=val&foo=val#t=boo",
			"http://example.com/?foo=val&bar=val#t=boo",
			false,
		},
		{
			"get",
			"GET",
			"example.com?foo=val&bar=val&n=another#t=boo",
			"http://example.com?bar=val&foo=val#t=boo",
			true,
		},
		{
			"GET",
			"GET",
			"example.com/?foo=val&bar=val#t=bo",
			"http://example.com/?foo=val&bar=val#t=boo",
			true,
		},
		{
			"get",
			"POST",
			"http://example.com",
			"http://example.com",
			true,
		},
	}

	for _, testcase := range testcases {
		stub := NewStubRequest(
			testcase.method,
			testcase.url,
			NewStringResponder(200, "ok"),
		)

		req, err := http.NewRequest(testcase.requestMethod, testcase.requestURL, nil)
		if err != nil {
			t.Fatalf("Unexpected error, got %#v", err)
		}

		err = stub.Matches(req)
		if testcase.expectedErr {
			if err == nil {
				t.Errorf("Didn't get error response when expected one: %s", testcase.url)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error, got %#v", err)
			}
		}
	}
}

func TestMatchesWithHeader(t *testing.T) {
	testcases := []struct {
		method         string
		stubURL        string
		stubHeaders    *http.Header
		requestURL     string
		requestHeaders http.Header
		expectedErr    bool
	}{
		{
			"get",
			"http://example.com",
			&http.Header{
				"X-ApiKey": []string{"api-key"},
			},
			"http://example.com",
			http.Header{
				http.CanonicalHeaderKey("X-ApiKey"): []string{"api-key"},
			},
			false,
		},
		{
			"get",
			"http://example.com",
			&http.Header{
				"X-ApiKey": []string{"api-key"},
			},
			"http://example.com",
			http.Header{
				http.CanonicalHeaderKey("X-ApiKey"): []string{"another-api-key"},
			},
			true,
		},
		{
			"get",
			"http://example.com",
			&http.Header{
				"X-ApiKey": []string{"api-key"},
				"Accept":   []string{"application/json"},
			},
			"http://example.com",
			http.Header{
				http.CanonicalHeaderKey("X-ApiKey"): []string{"api-key"},
			},
			true,
		},
	}

	for _, testcase := range testcases {
		stub := NewStubRequest(
			testcase.method,
			testcase.stubURL,
			NewStringResponder(200, "ok"),
		).WithHeader(testcase.stubHeaders)

		req, err := http.NewRequest("GET", testcase.requestURL, nil)
		if err != nil {
			t.Fatalf("Unexpected error, got %#v", err)
		}

		req.Header = testcase.requestHeaders

		err = stub.Matches(req)

		if testcase.expectedErr && err == nil {
			t.Errorf("Expected error, got none for %s", testcase.stubURL)
		} else if !testcase.expectedErr && err != nil {
			t.Errorf("Unexpected error, got '%#v' for %s", err, testcase.stubURL)
		}
	}
}

func TestRequestWithBody(t *testing.T) {
	testcases := []struct {
		method      string
		stubURL     string
		body        io.Reader
		requestURL  string
		requestBody io.Reader
		expectedErr bool
	}{
		{
			"POST",
			"http://example.com",
			bytes.NewBufferString("foo=val"),
			"http://example.com",
			bytes.NewBufferString("foo=val"),
			false,
		},
		{
			"POST",
			"http://example.com",
			bytes.NewBufferString("foo=val"),
			"http://example.com",
			bytes.NewBufferString("bar=val"),
			true,
		},
	}

	for _, testcase := range testcases {
		stub := NewStubRequest(
			testcase.method,
			testcase.stubURL,
			NewStringResponder(200, "ok"),
		).WithBody(testcase.body)

		req, err := http.NewRequest(testcase.method, testcase.requestURL, testcase.requestBody)
		if err != nil {
			t.Fatalf("Unexpected error, got %#v", err)
		}

		err = stub.Matches(req)

		if testcase.expectedErr && err == nil {
			t.Errorf("Expected error, got none for %s", testcase.stubURL)
		} else if !testcase.expectedErr && err != nil {
			t.Errorf("Unexpected error, got '%#v' for %s", err, testcase.stubURL)
		}
	}
}

func TestStubbedRequestStringer(t *testing.T) {
	testcases := []struct {
		method   string
		url      string
		header   *http.Header
		expected string
	}{
		{
			method:   "GET",
			url:      "http://example.com",
			expected: "GET http://example.com",
		},
		{
			method:   "POST",
			url:      "http://example.com/resource",
			expected: "POST http://example.com/resource",
		},
		{
			method: "GET",
			url:    "http://example.com",
			header: &http.Header{
				"X-ApiKey": []string{"api-key"},
			},
			expected: "GET http://example.com with headers &map[X-ApiKey:[api-key]]",
		},
	}

	for _, testcase := range testcases {
		stub := NewStubRequest(
			testcase.method,
			testcase.url,
			NewStringResponder(200, "ok"),
		).WithHeader(testcase.header)

		if stub.String() != testcase.expected {
			t.Errorf("Unexpected response, expected '%s', got '%s'", testcase.expected, stub.String())
		}
	}
}
