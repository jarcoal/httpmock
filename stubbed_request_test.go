package httpmock

import (
	//"bytes"
	//"io"
	"net/http"
	"testing"
)

func TestMatches(t *testing.T) {
	testcases := []struct {
		method      string
		url         string
		requestURL  string
		expectedErr bool
	}{
		{
			"get",
			"http://example.com",
			"http://example.com",
			false,
		},
		{
			"get",
			"ExAmPlE.com?foo=val&bar=val#t=boo",
			"http://example.com?bar=val&foo=val#t=boo",
			false,
		},
		{
			"Get",
			"http://example.com:80/?bar=val&foo=val#t=boo",
			"http://example.com/?foo=val&bar=val#t=boo",
			false,
		},
		{
			"get",
			"example.com?foo=val&bar=val&n=another#t=boo",
			"http://example.com?bar=val&foo=val#t=boo",
			true,
		},
		{
			"GET",
			"example.com/?foo=val&bar=val#t=bo",
			"http://example.com/?foo=val&bar=val#t=boo",
			true,
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

//func TestMatchesWithHeader(t *testing.T) {
//	testcases := []struct {
//		method         string
//		stubURL        string
//		stubHeaders    *http.Header
//		requestURL     string
//		requestHeaders http.Header
//		match          bool
//	}{
//		{
//			"get",
//			"http://example.com",
//			&http.Header{
//				"X-ApiKey": []string{"api-key"},
//			},
//			"http://example.com",
//			http.Header{
//				"X-ApiKey": []string{"api-key"},
//			},
//			true,
//		},
//		{
//			"get",
//			"http://example.com",
//			&http.Header{
//				"X-ApiKey": []string{"api-key"},
//			},
//			"http://example.com",
//			http.Header{
//				"X-ApiKey": []string{"another-api-key"},
//			},
//			false,
//		},
//		{
//			"get",
//			"http://example.com",
//			&http.Header{
//				"X-ApiKey": []string{"api-key"},
//				"Accept":   []string{"application/json"},
//			},
//			"http://example.com",
//			http.Header{
//				"X-ApiKey": []string{"api-key"},
//			},
//			false,
//		},
//	}
//
//	for _, testcase := range testcases {
//		stub := NewStubRequest(
//			testcase.method,
//			testcase.stubURL,
//			NewStringResponder(200, "ok"),
//		).WithHeader(testcase.stubHeaders)
//
//		req, err := http.NewRequest("GET", testcase.requestURL, nil)
//		if err != nil {
//			t.Fatalf("Unexpected error, got %#v", err)
//		}
//
//		req.Header = testcase.requestHeaders
//
//		if stub.Matches(req) != testcase.match {
//			t.Errorf("Unexpected result expected '%#v', got '%#v' for %s", testcase.match, stub.Matches(req), testcase.stubURL)
//		}
//	}
//}
//
//func TestRequestWithBody(t *testing.T) {
//	testcases := []struct {
//		method      string
//		stubURL     string
//		body        io.Reader
//		requestURL  string
//		requestBody io.Reader
//		match       bool
//	}{
//		{
//			"POST",
//			"http://example.com",
//			bytes.NewBufferString("foo=val"),
//			"http://example.com",
//			bytes.NewBufferString("foo=val"),
//			true,
//		},
//		{
//			"POST",
//			"http://example.com",
//			bytes.NewBufferString("foo=val"),
//			"http://example.com",
//			bytes.NewBufferString("bar=val"),
//			false,
//		},
//	}
//
//	for _, testcase := range testcases {
//		stub := NewStubRequest(
//			testcase.method,
//			testcase.stubURL,
//			NewStringResponder(200, "ok"),
//		).WithBody(testcase.body)
//
//		req, err := http.NewRequest(testcase.method, testcase.requestURL, testcase.requestBody)
//		if err != nil {
//			t.Fatalf("Unexpected error, got %#v", err)
//		}
//
//		if stub.Matches(req) != testcase.match {
//			t.Errorf("Unexpected result expected '%#v', got '%#v' for %s", testcase.match, stub.Matches(req), testcase.stubURL)
//		}
//	}
//}
