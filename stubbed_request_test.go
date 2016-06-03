package httpmock

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

func TestMatches(t *testing.T) {
	testcases := []struct {
		method     string
		url        string
		requestURL string
		match      bool
	}{
		{
			"get",
			"http://example.com",
			"http://example.com",
			true,
		},
		{
			"get",
			"ExAmPlE.com?foo=val&bar=val#t=boo",
			"http://example.com?bar=val&foo=val#t=boo",
			true,
		},
		{
			"Get",
			"http://example.com:80/?bar=val&foo=val#t=boo",
			"http://example.com/?foo=val&bar=val#t=boo",
			true,
		},
		{
			"get",
			"example.com?foo=val&bar=val&n=another#t=boo",
			"http://example.com?bar=val&foo=val#t=boo",
			false,
		},
		{
			"GET",
			"example.com/?foo=val&bar=val#t=bo",
			"http://example.com/?foo=val&bar=val#t=boo",
			false,
		},
	}

	for _, testcase := range testcases {
		stub := NewStubRequest(
			testcase.method,
			testcase.url,
			NewStringResponder(200, "ok"),
		)

		req, err := http.NewRequest("GET", testcase.requestURL, nil)
		if err != nil {
			t.Fatalf("Unexpected error, got %#v", err)
		}

		if stub.Matches(req) != testcase.match {
			t.Errorf("Unexpected result expected '%#v', got '%#v' for %s", testcase.match, stub.Matches(req), testcase.url)
		}
	}
}

func TestMatchesWithHeaders(t *testing.T) {
	testcases := []struct {
		method         string
		stubUrl        string
		stubHeaders    *http.Header
		requestURL     string
		requestHeaders http.Header
		match          bool
	}{
		{
			"get",
			"http://example.com",
			&http.Header{
				"X-ApiKey": []string{"api-key"},
			},
			"http://example.com",
			http.Header{
				"X-ApiKey": []string{"api-key"},
			},
			true,
		},
		{
			"get",
			"http://example.com",
			&http.Header{
				"X-ApiKey": []string{"api-key"},
			},
			"http://example.com",
			http.Header{
				"X-ApiKey": []string{"another-api-key"},
			},
			false,
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
				"X-ApiKey": []string{"api-key"},
			},
			false,
		},
	}

	for _, testcase := range testcases {
		stub := NewStubRequest(
			testcase.method,
			testcase.stubUrl,
			NewStringResponder(200, "ok"),
		).WithHeaders(testcase.stubHeaders)

		req, err := http.NewRequest("GET", testcase.requestURL, nil)
		if err != nil {
			t.Fatalf("Unexpected error, got %#v", err)
		}

		req.Header = testcase.requestHeaders

		if stub.Matches(req) != testcase.match {
			t.Errorf("Unexpected result expected '%#v', got '%#v' for %s", testcase.match, stub.Matches(req), testcase.stubUrl)
		}
	}
}

func TestRequestWithBody(t *testing.T) {
	testcases := []struct {
		method      string
		stubUrl     string
		body        io.Reader
		requestURL  string
		requestBody io.Reader
		match       bool
	}{
		{
			"POST",
			"http://example.com",
			bytes.NewBufferString("foo=val"),
			"http://example.com",
			bytes.NewBufferString("foo=val"),
			true,
		},
		{
			"POST",
			"http://example.com",
			bytes.NewBufferString("foo=val"),
			"http://example.com",
			bytes.NewBufferString("bar=val"),
			false,
		},
	}

	for _, testcase := range testcases {
		stub := NewStubRequest(
			testcase.method,
			testcase.stubUrl,
			NewStringResponder(200, "ok"),
		).WithBody(testcase.body)

		req, err := http.NewRequest(testcase.method, testcase.requestURL, testcase.requestBody)
		if err != nil {
			t.Fatalf("Unexpected error, got %#v", err)
		}

		if stub.Matches(req) != testcase.match {
			t.Errorf("Unexpected result expected '%#v', got '%#v' for %s", testcase.match, stub.Matches(req), testcase.stubUrl)
		}
	}
}
