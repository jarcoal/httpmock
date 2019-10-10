package httpmock

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

var testURL = "http://www.example.com/"

func assertBody(t *testing.T, resp *http.Response, expected string) {
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	got := string(data)

	if got != expected {
		t.Errorf("Expected body: %#v, got %#v", expected, got)
	}
}

func TestRouteKey(t *testing.T) {
	got, expected := noResponder.String(), "NO_RESPONDER"
	if got != expected {
		t.Errorf("got: %v, expected: %v", got, expected)
	}

	got, expected = routeKey{Method: http.MethodGet, URL: "/foo"}.String(), "GET /foo"
	if got != expected {
		t.Errorf("got: %v, expected: %v", got, expected)
	}
}

func TestMockTransport(t *testing.T) {
	Activate()
	defer Deactivate()

	url := "https://github.com/"
	body := `["hello world"]` + "\n"

	RegisterResponder(http.MethodGet, url, NewStringResponder(http.StatusOK, body))

	// Read it as a simple string (ioutil.ReadAll will trigger io.EOF)
	func() {
		resp, err := http.Get(url)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		if string(data) != body {
			t.FailNow()
		}

		// the http client wraps our NoResponderFound error, so we just try and match on text
		if _, err := http.Get(testURL); !strings.Contains(err.Error(),
			NoResponderFound.Error()) {

			t.Fatal(err)
		}
	}()

	// Do it again, but twice with json decoder (json Decode will not
	// reach EOF, but Close is called as the JSON response is complete)
	for i := 0; i < 2; i++ {
		func() {
			resp, err := http.Get(url)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			var res []string
			err = json.NewDecoder(resp.Body).Decode(&res)
			if err != nil {
				t.Fatal(err)
			}

			if len(res) != 1 || res[0] != "hello world" {
				t.Fatalf(`%v read instead of ["hello world"]`, res)
			}
		}()
	}
}

// We should be able to find GET handlers when using an http.Request with a
// default (zero-value) .Method.
func TestMockTransportDefaultMethod(t *testing.T) {
	Activate()
	defer Deactivate()

	const urlString = "https://github.com/"
	url, err := url.Parse(urlString)
	if err != nil {
		t.Fatal(err)
	}
	body := "hello world"

	RegisterResponder(http.MethodGet, urlString, NewStringResponder(http.StatusOK, body))

	req := &http.Request{
		URL: url,
		// Note: Method unspecified (zero-value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != body {
		t.FailNow()
	}
}

func TestMockTransportReset(t *testing.T) {
	DeactivateAndReset()

	if len(DefaultTransport.responders) > 0 {
		t.Fatal("expected no responders at this point")
	}

	RegisterResponder(http.MethodGet, testURL, nil)

	if len(DefaultTransport.responders) != 1 {
		t.Fatal("expected one responder")
	}

	Reset()

	if len(DefaultTransport.responders) > 0 {
		t.Fatal("expected no responders as they were just reset")
	}
}

func TestMockTransportNoResponder(t *testing.T) {
	Activate()
	defer DeactivateAndReset()

	Reset()

	if DefaultTransport.noResponder != nil {
		t.Fatal("expected noResponder to be nil")
	}

	if _, err := http.Get(testURL); err == nil {
		t.Fatal("expected to receive a connection error due to lack of responders")
	}

	RegisterNoResponder(NewStringResponder(http.StatusOK, "hello world"))

	resp, err := http.Get(testURL)
	if err != nil {
		t.Fatal("expected request to succeed")
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != "hello world" {
		t.Fatal("expected body to be 'hello world'")
	}
}

func TestMockTransportQuerystringFallback(t *testing.T) {
	Activate()
	defer DeactivateAndReset()

	// register the testURL responder
	RegisterResponder(http.MethodGet, testURL, NewStringResponder(http.StatusOK, "hello world"))

	for _, suffix := range []string{"?", "?hello=world", "?hello=world#foo", "?hello=world&hello=all", "#foo"} {
		reqURL := testURL + suffix

		// make a request for the testURL with a querystring
		resp, err := http.Get(reqURL)
		if err != nil {
			t.Fatalf("expected request %s to succeed", reqURL)
		}

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("%s error: %s", reqURL, err)
		}

		if string(data) != "hello world" {
			t.Fatalf("expected body of %s to be 'hello world'", reqURL)
		}
	}
}

func TestMockTransportPathOnlyFallback(t *testing.T) {
	// Just in case a panic occurs
	defer DeactivateAndReset()

	for _, test := range []struct {
		Responder string
		Paths     []string
	}{
		{
			// unsorted query string matches exactly
			Responder: "/hello/world?query=string&abc=zz#fragment",
			Paths: []string{
				testURL + "hello/world?query=string&abc=zz#fragment",
			},
		},
		{
			// sorted query string matches all cases
			Responder: "/hello/world?abc=zz&query=string#fragment",
			Paths: []string{
				testURL + "hello/world?query=string&abc=zz#fragment",
				testURL + "hello/world?abc=zz&query=string#fragment",
			},
		},
		{
			// unsorted query string matches exactly
			Responder: "/hello/world?query=string&abc=zz",
			Paths: []string{
				testURL + "hello/world?query=string&abc=zz",
			},
		},
		{
			// sorted query string matches all cases
			Responder: "/hello/world?abc=zz&query=string",
			Paths: []string{
				testURL + "hello/world?query=string&abc=zz",
				testURL + "hello/world?abc=zz&query=string",
			},
		},
		{
			// unsorted query string matches exactly
			Responder: "/hello/world?query=string&query=string2&abc=zz",
			Paths: []string{
				testURL + "hello/world?query=string&query=string2&abc=zz",
			},
		},
		// sorted query string matches all cases
		{
			Responder: "/hello/world?abc=zz&query=string&query=string2",
			Paths: []string{
				testURL + "hello/world?query=string&query=string2&abc=zz",
				testURL + "hello/world?query=string2&query=string&abc=zz",
				testURL + "hello/world?abc=zz&query=string2&query=string",
			},
		},
		{
			Responder: "/hello/world?query",
			Paths: []string{
				testURL + "hello/world?query",
			},
		},
		{
			Responder: "/hello/world?query&abc",
			Paths: []string{
				testURL + "hello/world?query&abc",
				// testURL + "hello/world?abc&query" won' work as "=" is needed, see below
			},
		},
		{
			// In case the sorting does not matter for received params without
			// values, we must register params with "="
			Responder: "/hello/world?abc=&query=",
			Paths: []string{
				testURL + "hello/world?query&abc",
				testURL + "hello/world?abc&query",
			},
		},
		{
			Responder: "/hello/world#fragment",
			Paths: []string{
				testURL + "hello/world#fragment",
			},
		},
		{
			Responder: "/hello/world",
			Paths: []string{
				testURL + "hello/world?query=string&abc=zz#fragment",
				testURL + "hello/world?query=string&abc=zz",
				testURL + "hello/world#fragment",
				testURL + "hello/world",
			},
		},
		// Regexp cases
		{
			Responder: `=~^http://.*/hello/.*ld\z`,
			Paths: []string{
				testURL + "hello/world?query=string&abc=zz#fragment",
				testURL + "hello/world?query=string&abc=zz",
				testURL + "hello/world#fragment",
				testURL + "hello/world",
			},
		},
		{
			Responder: `=~^http://.*/hello/.*ld(\z|[?#])`,
			Paths: []string{
				testURL + "hello/world?query=string&abc=zz#fragment",
				testURL + "hello/world?query=string&abc=zz",
				testURL + "hello/world#fragment",
				testURL + "hello/world",
			},
		},
		{
			Responder: `=~^/hello/.*ld\z`,
			Paths: []string{
				testURL + "hello/world?query=string&abc=zz#fragment",
				testURL + "hello/world?query=string&abc=zz",
				testURL + "hello/world#fragment",
				testURL + "hello/world",
			},
		},
		{
			Responder: `=~^/hello/.*ld(\z|[?#])`,
			Paths: []string{
				testURL + "hello/world?query=string&abc=zz#fragment",
				testURL + "hello/world?query=string&abc=zz",
				testURL + "hello/world#fragment",
				testURL + "hello/world",
			},
		},
		{
			Responder: `=~abc=zz`,
			Paths: []string{
				testURL + "hello/world?query=string&abc=zz#fragment",
				testURL + "hello/world?query=string&abc=zz",
			},
		},
	} {
		Activate()

		// register the responder
		RegisterResponder(http.MethodGet, test.Responder, NewStringResponder(http.StatusOK, "hello world"))

		for _, reqURL := range test.Paths {
			// make a request for the testURL with a querystring
			resp, err := http.Get(reqURL)
			if err != nil {
				t.Fatalf("%s: expected request %s to succeed", test.Responder, reqURL)
			}

			data, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("%s: %s error: %s", test.Responder, reqURL, err)
			}

			if string(data) != "hello world" {
				t.Fatalf("%s: expected body of %s to be 'hello world'", test.Responder, reqURL)
			}
		}

		DeactivateAndReset()
	}
}

type dummyTripper struct{}

func (d *dummyTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, nil
}

func TestMockTransportInitialTransport(t *testing.T) {
	DeactivateAndReset()

	tripper := &dummyTripper{}
	http.DefaultTransport = tripper

	Activate()

	if http.DefaultTransport == tripper {
		t.Fatal("expected http.DefaultTransport to be a mock transport")
	}

	Deactivate()

	if http.DefaultTransport != tripper {
		t.Fatal("expected http.DefaultTransport to be dummy")
	}
}

func TestMockTransportNonDefault(t *testing.T) {
	// create a custom http client w/ custom Roundtripper
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   60 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 60 * time.Second,
		},
	}

	// activate mocks for the client
	ActivateNonDefault(client)
	defer DeactivateAndReset()

	body := "hello world!"

	RegisterResponder(http.MethodGet, testURL, NewStringResponder(http.StatusOK, body))

	req, err := http.NewRequest(http.MethodGet, testURL, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != body {
		t.FailNow()
	}
}

func TestMockTransportRespectsCancel(t *testing.T) {
	Activate()
	defer DeactivateAndReset()

	const (
		cancelNone = iota
		cancelReq
		cancelCtx
	)

	cases := []struct {
		withCancel   int
		cancelNow    bool
		withPanic    bool
		expectedBody string
		expectedErr  error
	}{
		// No cancel specified at all. Falls back to normal behavior
		{cancelNone, false, false, "hello world", nil},

		// Cancel returns error
		{cancelReq, true, false, "", errors.New("request canceled")},

		// Cancel via context returns error
		{cancelCtx, true, false, "", errors.New("context canceled")},

		// Request can be cancelled but it is not cancelled.
		{cancelReq, false, false, "hello world", nil},

		// Request can be cancelled but it is not cancelled.
		{cancelCtx, false, false, "hello world", nil},

		// Panic in cancelled request is handled
		{cancelReq, false, true, "", errors.New(`panic in responder: got "oh no"`)},

		// Panic in cancelled request is handled
		{cancelCtx, false, true, "", errors.New(`panic in responder: got "oh no"`)},
	}

	for _, c := range cases {
		Reset()
		if c.withPanic {
			RegisterResponder(http.MethodGet, testURL, func(r *http.Request) (*http.Response, error) {
				time.Sleep(10 * time.Millisecond)
				panic("oh no")
			})
		} else {
			RegisterResponder(http.MethodGet, testURL, func(r *http.Request) (*http.Response, error) {
				time.Sleep(10 * time.Millisecond)
				return NewStringResponse(http.StatusOK, "hello world"), nil
			})
		}

		req, err := http.NewRequest(http.MethodGet, testURL, nil)
		if err != nil {
			t.Fatal(err)
		}

		switch c.withCancel {
		case cancelReq:
			cancel := make(chan struct{}, 1)
			req.Cancel = cancel // nolint: staticcheck
			if c.cancelNow {
				cancel <- struct{}{}
			}
		case cancelCtx:
			ctx, cancel := context.WithCancel(req.Context())
			req = req.WithContext(ctx)
			if c.cancelNow {
				cancel()
			} else {
				defer cancel() // avoid ctx leak
			}
		}

		resp, err := http.DefaultClient.Do(req)

		// If we expect an error but none was returned, it's fatal for this test...
		if err == nil && c.expectedErr != nil {
			t.Fatal("Error should not be nil")
		}

		if err != nil {
			got := err.(*url.Error)
			// Do not use reflect.DeepEqual as go 1.13 includes stack frames
			// into errors issued by errors.New()
			if c.expectedErr == nil || got.Err.Error() != c.expectedErr.Error() {
				t.Errorf("Expected error: %v, got: %v", c.expectedErr, got.Err)
			}
		}

		if c.expectedBody != "" {
			assertBody(t, resp, c.expectedBody)
		}
	}
}

func TestMockTransportRespectsTimeout(t *testing.T) {
	timeout := time.Millisecond
	client := &http.Client{
		Timeout: timeout,
	}

	ActivateNonDefault(client)
	defer DeactivateAndReset()

	RegisterResponder(
		http.MethodGet, testURL,
		func(r *http.Request) (*http.Response, error) {
			time.Sleep(100 * timeout)
			return NewStringResponse(http.StatusOK, ""), nil
		},
	)

	_, err := client.Get(testURL)
	if err == nil {
		t.Fail()
	}
}

func TestMockTransportCallCount(t *testing.T) {
	Reset()
	Activate()
	defer Deactivate()

	const (
		url  = "https://github.com/path?b=1&a=2"
		url2 = "https://gitlab.com/"
	)

	RegisterResponder(http.MethodGet, url, NewStringResponder(http.StatusOK, "body"))
	RegisterResponder(http.MethodPost, "=~gitlab", NewStringResponder(http.StatusOK, "body"))

	_, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}

	buff := new(bytes.Buffer)
	json.NewEncoder(buff).Encode("{}") // nolint: errcheck
	_, err = http.Post(url2, "application/json", buff)
	if err != nil {
		t.Fatal(err)
	}

	_, err = http.Get(url)
	if err != nil {
		t.Fatal(err)
	}

	totalCallCount := GetTotalCallCount()
	if totalCallCount != 3 {
		t.Fatalf("did not track the total count of calls correctly. expected it to be 3, but it was %v", totalCallCount)
	}

	info := GetCallCountInfo()
	expectedInfo := map[string]int{
		"GET " + url: 2,
		// Regexp match generates 2 entries:
		"POST " + url2:  1, // the matched call
		"POST =~gitlab": 1, // the regexp responder
	}

	if !reflect.DeepEqual(info, expectedInfo) {
		t.Fatalf("did not correctly track the call count info. expected it to be \n %+v \n but it was \n %+v \n", expectedInfo, info)
	}

	Reset()

	afterResetTotalCallCount := GetTotalCallCount()
	if afterResetTotalCallCount != 0 {
		t.Fatalf("did not reset the total count of calls correctly. expected it to be 0 after reset, but it was %v", afterResetTotalCallCount)
	}
}

func TestRegisterResponderWithQuery(t *testing.T) {
	// Just in case a panic occurs
	defer DeactivateAndReset()

	// create a custom http client w/ custom Roundtripper
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   60 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 60 * time.Second,
		},
	}

	body := "hello world!"
	testURLPath := "http://acme.test/api"

	for _, test := range []struct {
		URL     string
		Queries []interface{}
		URLs    []string
	}{
		{
			Queries: []interface{}{
				map[string]string{"a": "1", "b": "2"},
				"a=1&b=2",
				"b=2&a=1",
				url.Values{"a": []string{"1"}, "b": []string{"2"}},
			},
			URLs: []string{
				"http://acme.test/api?a=1&b=2",
				"http://acme.test/api?b=2&a=1",
			},
		},
		{
			Queries: []interface{}{
				url.Values{
					"a": []string{"3", "2", "1"},
					"b": []string{"4", "2"},
					"c": []string{""}, // is the net/url way to record params without values
					// Test:
					//   u, _ := url.Parse("/hello/world?query")
					//   fmt.Printf("%d<%s>\n", len(u.Query()["query"]), u.Query()["query"][0])
					//   // prints "1<>"
				},
				"a=1&b=2&a=3&c&b=4&a=2",
				"b=2&a=1&c=&b=4&a=2&a=3",
				nil,
			},
			URLs: []string{
				testURLPath + "?a=1&b=2&a=3&c&b=4&a=2",
				testURLPath + "?a=1&b=2&a=3&c=&b=4&a=2",
				testURLPath + "?b=2&a=1&c=&b=4&a=2&a=3",
				testURLPath + "?b=2&a=1&c&b=4&a=2&a=3",
			},
		},
	} {
		for _, query := range test.Queries {
			ActivateNonDefault(client)
			RegisterResponderWithQuery(http.MethodGet, testURLPath, query, NewStringResponder(http.StatusOK, body))

			for _, url := range test.URLs {
				req, err := http.NewRequest(http.MethodGet, url, nil)
				if err != nil {
					t.Fatal(err)
				}
				resp, err := client.Do(req)
				if err != nil {
					t.Fatal(err)
				}
				data, err := ioutil.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					t.Fatal(err)
				}
				if string(data) != body {
					t.Fatalf("query=%v URL=%s: %s ≠ %s", query, url, string(data), body)
				}
			}

			DeactivateAndReset()
		}
	}
}

func TestRegisterResponderWithQueryPanic(t *testing.T) {
	resp := NewStringResponder(http.StatusOK, "hello world!")

	for _, test := range []struct {
		Path        string
		Query       interface{}
		PanicPrefix string
	}{
		{
			Path:        "foobar",
			Query:       "%",
			PanicPrefix: "RegisterResponderWithQuery bad query string: ",
		},
		{
			Path:        "foobar",
			Query:       1234,
			PanicPrefix: "RegisterResponderWithQuery bad query type int. Only url.Values, map[string]string and string are allowed",
		},
		{
			Path:        `=~regexp.*\z`,
			Query:       "",
			PanicPrefix: `path begins with "=~", RegisterResponder should be used instead of RegisterResponderWithQuery`,
		},
	} {
		var (
			didntPanic bool
			panicVal   interface{}
		)
		func() {
			defer func() {
				panicVal = recover()
			}()

			RegisterResponderWithQuery(http.MethodGet, test.Path, test.Query, resp)
			didntPanic = true
		}()

		if didntPanic {
			t.Fatalf("RegisterResponderWithQuery + query=%v did not panic", test.Query)
		}

		panicStr, ok := panicVal.(string)
		if !ok || !strings.HasPrefix(panicStr, test.PanicPrefix) {
			t.Fatalf(`RegisterResponderWithQuery + query=%v panic="%v" expected prefix="%v"`,
				test.Query, panicVal, test.PanicPrefix)
		}
	}
}

func TestRegisterRegexpResponder(t *testing.T) {
	Activate()
	defer DeactivateAndReset()

	rx := regexp.MustCompile("ex.mple")

	RegisterRegexpResponder(http.MethodGet, rx, NewStringResponder(http.StatusOK, "first"))
	// Overwrite responder
	RegisterRegexpResponder(http.MethodGet, rx, NewStringResponder(http.StatusOK, "second"))

	resp, err := http.Get(testURL)
	if err != nil {
		t.Fatalf("expected request %s to succeed", testURL)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("%s error: %s", testURL, err)
	}

	if string(data) != "second" {
		t.Fatalf("expected body of %s to be 'hello world'", testURL)
	}
}

func TestSubmatches(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/foo/bar", nil)
	if err != nil {
		t.Fatal(err)
	}

	var req2 *http.Request

	t.Run("setSubmatches", func(t *testing.T) {
		req2 = setSubmatches(req, nil)
		if req2 != req {
			t.Error("setSubmatches(req, nil) should return the same request")
		}

		req2 = setSubmatches(req, []string{})
		if req2 != req {
			t.Error("setSubmatches(req, []string{}) should return the same request")
		}

		req2 = setSubmatches(req, []string{"foo", "123", "-123", "12.3"})
		if req2 == req {
			t.Error("setSubmatches(req, []string{...}) should NOT return the same request")
		}
	})

	t.Run("GetSubmatch", func(t *testing.T) {
		_, err := GetSubmatch(req, 1)
		if err != ErrSubmatchNotFound {
			t.Errorf("Submatch should not be found in req: %v", err)
		}

		_, err = GetSubmatch(req2, 5)
		if err != ErrSubmatchNotFound {
			t.Errorf("Submatch #5 should not be found in req2: %v", err)
		}

		s, err := GetSubmatch(req2, 1)
		if err != nil {
			t.Errorf("GetSubmatch(req2, 1) failed: %v", err)
		}
		if s != "foo" {
			t.Errorf("GetSubmatch(req2, 1) failed, got: %v, expected: foo", s)
		}

		s, err = GetSubmatch(req2, 4)
		if err != nil {
			t.Errorf("GetSubmatch(req2, 4) failed: %v", err)
		}
		if s != "12.3" {
			t.Errorf("GetSubmatch(req2, 4) failed, got: %v, expected: 12.3", s)
		}

		s = MustGetSubmatch(req2, 4)
		if s != "12.3" {
			t.Errorf("GetSubmatch(req2, 4) failed, got: %v, expected: 12.3", s)
		}
	})

	t.Run("GetSubmatchAsInt", func(t *testing.T) {
		_, err := GetSubmatchAsInt(req, 1)
		if err != ErrSubmatchNotFound {
			t.Errorf("Submatch should not be found in req: %v", err)
		}

		_, err = GetSubmatchAsInt(req2, 4) // not an int
		if err == nil || err == ErrSubmatchNotFound {
			t.Errorf("Submatch should not be an int64: %v", err)
		}

		i, err := GetSubmatchAsInt(req2, 3)
		if err != nil {
			t.Errorf("GetSubmatchAsInt(req2, 3) failed: %v", err)
		}
		if i != -123 {
			t.Errorf("GetSubmatchAsInt(req2, 3) failed, got: %d, expected: -123", i)
		}

		i = MustGetSubmatchAsInt(req2, 3)
		if i != -123 {
			t.Errorf("MustGetSubmatchAsInt(req2, 3) failed, got: %d, expected: -123", i)
		}
	})

	t.Run("GetSubmatchAsUint", func(t *testing.T) {
		_, err := GetSubmatchAsUint(req, 1)
		if err != ErrSubmatchNotFound {
			t.Errorf("Submatch should not be found in req: %v", err)
		}

		_, err = GetSubmatchAsUint(req2, 3) // not a uint
		if err == nil || err == ErrSubmatchNotFound {
			t.Errorf("Submatch should not be an uint64: %v", err)
		}

		u, err := GetSubmatchAsUint(req2, 2)
		if err != nil {
			t.Errorf("GetSubmatchAsUint(req2, 2) failed: %v", err)
		}
		if u != 123 {
			t.Errorf("GetSubmatchAsUint(req2, 2) failed, got: %d, expected: 123", u)
		}

		u = MustGetSubmatchAsUint(req2, 2)
		if u != 123 {
			t.Errorf("MustGetSubmatchAsUint(req2, 2) failed, got: %d, expected: 123", u)
		}
	})

	t.Run("GetSubmatchAsFloat", func(t *testing.T) {
		_, err := GetSubmatchAsFloat(req, 1)
		if err != ErrSubmatchNotFound {
			t.Errorf("Submatch should not be found in req: %v", err)
		}

		_, err = GetSubmatchAsFloat(req2, 1) // not a float
		if err == nil || err == ErrSubmatchNotFound {
			t.Errorf("Submatch should not be an float64: %v", err)
		}

		f, err := GetSubmatchAsFloat(req2, 4)
		if err != nil {
			t.Errorf("GetSubmatchAsFloat(req2, 4) failed: %v", err)
		}
		if f != 12.3 {
			t.Errorf("GetSubmatchAsFloat(req2, 4) failed, got: %f, expected: 12.3", f)
		}

		f = MustGetSubmatchAsFloat(req2, 4)
		if f != 12.3 {
			t.Errorf("MustGetSubmatchAsFloat(req2, 4) failed, got: %f, expected: 12.3", f)
		}
	})

	t.Run("GetSubmatch* panics", func(t *testing.T) {
		for _, test := range []struct {
			Name        string
			Fn          func()
			PanicPrefix string
		}{
			{
				Name:        "GetSubmatch & n < 1",
				Fn:          func() { GetSubmatch(req, 0) }, // nolint: errcheck
				PanicPrefix: "getting submatches starts at 1, not 0",
			},
			{
				Name:        "MustGetSubmatch",
				Fn:          func() { MustGetSubmatch(req, 1) },
				PanicPrefix: "GetSubmatch failed: " + ErrSubmatchNotFound.Error(),
			},
			{
				Name:        "MustGetSubmatchAsInt",
				Fn:          func() { MustGetSubmatchAsInt(req2, 4) }, // not an int
				PanicPrefix: "GetSubmatchAsInt failed: ",
			},
			{
				Name:        "MustGetSubmatchAsUint",
				Fn:          func() { MustGetSubmatchAsUint(req2, 3) }, // not a uint
				PanicPrefix: "GetSubmatchAsUint failed: ",
			},
			{
				Name:        "GetSubmatchAsFloat",
				Fn:          func() { MustGetSubmatchAsFloat(req2, 1) }, // not a float
				PanicPrefix: "GetSubmatchAsFloat failed: ",
			},
		} {
			var (
				didntPanic bool
				panicVal   interface{}
			)
			func() {
				defer func() { panicVal = recover() }()
				test.Fn()
				didntPanic = true
			}()

			if didntPanic {
				t.Errorf("%s did not panic", test.Name)
			}

			panicStr, ok := panicVal.(string)
			if !ok || !strings.HasPrefix(panicStr, test.PanicPrefix) {
				t.Errorf(`%s panic="%v" expected prefix="%v"`, test.Name, panicVal, test.PanicPrefix)
			}
		}
	})

	t.Run("Full test", func(t *testing.T) {
		Activate()
		defer DeactivateAndReset()

		var (
			id       uint64
			delta    float64
			deltaStr string
			inc      int64
		)
		RegisterResponder(http.MethodGet, `=~^/id/(\d+)\?delta=(\d+(?:\.\d*)?)&inc=(-?\d+)\z`,
			func(req *http.Request) (*http.Response, error) {
				id = MustGetSubmatchAsUint(req, 1)
				delta = MustGetSubmatchAsFloat(req, 2)
				deltaStr = MustGetSubmatch(req, 2)
				inc = MustGetSubmatchAsInt(req, 3)

				return NewStringResponse(http.StatusOK, "OK"), nil
			})

		resp, err := http.Get("http://example.tld/id/123?delta=1.2&inc=-5")
		if err != nil {
			t.Fatal(err)
		}
		assertBody(t, resp, "OK")

		// Check submatches
		if id != 123 {
			t.Errorf("seems MustGetSubmatchAsUint failed, got: %d, expected: 123", id)
		}
		if delta != 1.2 {
			t.Errorf("seems MustGetSubmatchAsFloat failed, got: %f, expected: 1.2", delta)
		}
		if deltaStr != "1.2" {
			t.Errorf("seems MustGetSubmatch failed, got: %v, expected: 1.2", deltaStr)
		}
		if inc != -5 {
			t.Errorf("seems MustGetSubmatchAsInt failed, got: %d, expected: 123", inc)
		}
	})
}

func TestCheckStackTracer(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://foo.bar/", nil)
	if err != nil {
		t.Fatal(err)
	}

	// no error
	gotErr := checkStackTracer(req, nil)
	if gotErr != nil {
		t.Errorf(`checkStackTracer(nil) should return nil, not %v`, gotErr)
	}

	// Classic error
	err = errors.New("error")
	gotErr = checkStackTracer(req, err)
	if err != gotErr {
		t.Errorf(`checkStackTracer(err) should return %v, not %v`, err, gotErr)
	}

	// stackTracer without customFn
	origErr := errors.New("foo")
	errTracer := stackTracer{
		err: origErr,
	}
	gotErr = checkStackTracer(req, errTracer)
	if gotErr != origErr {
		t.Errorf(`Returned error mismatch, expected: %v, got: %v`, origErr, gotErr)
	}

	// stackTracer with nil error & without customFn
	errTracer = stackTracer{}
	gotErr = checkStackTracer(req, errTracer)
	if gotErr != nil {
		t.Errorf(`Returned error mismatch, expected: nil, got: %v`, gotErr)
	}

	// stackTracer
	var mesg string
	errTracer = stackTracer{
		err: origErr,
		customFn: func(args ...interface{}) {
			mesg = args[0].(string)
		},
	}
	gotErr = checkStackTracer(req, errTracer)
	if !strings.HasPrefix(mesg, "foo\nCalled from ") || strings.HasSuffix(mesg, "\n") {
		t.Errorf(`mesg does not match "^foo\nCalled from .*[^\n]\z", it is "` + mesg + `"`)
	}
	if gotErr != origErr {
		t.Errorf(`Returned error mismatch, expected: %v, got: %v`, origErr, gotErr)
	}

	// stackTracer with nil error but customFn
	mesg = ""
	errTracer = stackTracer{
		customFn: func(args ...interface{}) {
			mesg = args[0].(string)
		},
	}
	gotErr = checkStackTracer(req, errTracer)
	if !strings.HasPrefix(mesg, "GET http://foo.bar/\nCalled from ") || strings.HasSuffix(mesg, "\n") {
		t.Errorf(`mesg does not match "^foo\nCalled from .*[^\n]\z", it is "` + mesg + `"`)
	}
	if gotErr != nil {
		t.Errorf(`Returned error mismatch, expected: nil, got: %v`, gotErr)
	}

	// Full test using Trace() Responder
	Activate()
	defer Deactivate()

	const url = "https://foo.bar/"
	mesg = ""
	RegisterResponder(http.MethodGet, url,
		NewStringResponder(http.StatusOK, "{}").
			Trace(func(args ...interface{}) { mesg = args[0].(string) }))

	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != "{}" {
		t.FailNow()
	}

	// Check that first frame is the net/http.Get() call
	if !strings.HasPrefix(mesg, "GET https://foo.bar/\nCalled from net/http.Get()\n    at ") ||
		strings.HasSuffix(mesg, "\n") {
		t.Errorf("Bad mesg: <%v>", mesg)
	}
}
