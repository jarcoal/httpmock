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

func TestMockTransport(t *testing.T) {
	Activate()
	defer Deactivate()

	url := "https://github.com/"
	body := `["hello world"]` + "\n"

	RegisterResponder("GET", url, NewStringResponder(200, body))

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

	urlString := "https://github.com/"
	url, err := url.Parse(urlString)
	if err != nil {
		t.Fatal(err)
	}
	body := "hello world"

	RegisterResponder("GET", urlString, NewStringResponder(200, body))

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

	RegisterResponder("GET", testURL, nil)

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

	RegisterNoResponder(NewStringResponder(200, "hello world"))

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
	RegisterResponder("GET", testURL, NewStringResponder(200, "hello world"))

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

	for responder, paths := range map[string][]string{
		// unsorted query string matches exactly
		"/hello/world?query=string&abc=zz#fragment": {
			testURL + "hello/world?query=string&abc=zz#fragment",
		},
		// sorted query string matches all cases
		"/hello/world?abc=zz&query=string#fragment": {
			testURL + "hello/world?query=string&abc=zz#fragment",
			testURL + "hello/world?abc=zz&query=string#fragment",
		},
		// unsorted query string matches exactly
		"/hello/world?query=string&abc=zz": {
			testURL + "hello/world?query=string&abc=zz",
		},
		// sorted query string matches all cases
		"/hello/world?abc=zz&query=string": {
			testURL + "hello/world?query=string&abc=zz",
			testURL + "hello/world?abc=zz&query=string",
		},
		// unsorted query string matches exactly
		"/hello/world?query=string&query=string2&abc=zz": {
			testURL + "hello/world?query=string&query=string2&abc=zz",
		},
		// sorted query string matches all cases
		"/hello/world?abc=zz&query=string&query=string2": {
			testURL + "hello/world?query=string&query=string2&abc=zz",
			testURL + "hello/world?query=string2&query=string&abc=zz",
			testURL + "hello/world?abc=zz&query=string2&query=string",
		},
		"/hello/world?query": {
			testURL + "hello/world?query",
		},
		"/hello/world?query&abc": {
			testURL + "hello/world?query&abc",
			// testURL + "hello/world?abc&query" won' work as "=" is needed, see below
		},
		// In case the sorting does not matter for received params without
		// values, we must register params with "="
		"/hello/world?abc=&query=": {
			testURL + "hello/world?query&abc",
			testURL + "hello/world?abc&query",
		},
		"/hello/world#fragment": {
			testURL + "hello/world#fragment",
		},
		"/hello/world": {
			testURL + "hello/world?query=string&abc=zz#fragment",
			testURL + "hello/world?query=string&abc=zz",
			testURL + "hello/world#fragment",
			testURL + "hello/world",
		},
	} {
		Activate()

		// register the responder
		RegisterResponder("GET", responder, NewStringResponder(200, "hello world"))

		for _, reqURL := range paths {
			// make a request for the testURL with a querystring
			resp, err := http.Get(reqURL)
			if err != nil {
				t.Fatalf("%s: expected request %s to succeed", responder, reqURL)
			}

			data, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("%s: %s error: %s", responder, reqURL, err)
			}

			if string(data) != "hello world" {
				t.Fatalf("%s: expected body of %s to be 'hello world'", responder, reqURL)
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

	RegisterResponder("GET", testURL, NewStringResponder(200, body))

	req, err := http.NewRequest("GET", testURL, nil)
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
			RegisterResponder("GET", testURL, func(r *http.Request) (*http.Response, error) {
				time.Sleep(10 * time.Millisecond)
				panic("oh no")
			})
		} else {
			RegisterResponder("GET", testURL, func(r *http.Request) (*http.Response, error) {
				time.Sleep(10 * time.Millisecond)
				return NewStringResponse(http.StatusOK, "hello world"), nil
			})
		}

		req, err := http.NewRequest("GET", testURL, nil)
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
		"GET", testURL,
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

	url := "https://github.com/"
	url2 := "https://gitlab.com/"

	RegisterResponder("GET", url, NewStringResponder(200, "body"))
	RegisterResponder("POST", url2, NewStringResponder(200, "body"))

	_, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}

	buff := new(bytes.Buffer)
	json.NewEncoder(buff).Encode("{}") // nolint: errcheck
	_, err1 := http.Post(url2, "application/json", buff)
	if err1 != nil {
		t.Fatal(err1)
	}

	_, err2 := http.Get(url)
	if err2 != nil {
		t.Fatal(err2)
	}

	totalCallCount := GetTotalCallCount()
	if totalCallCount != 3 {
		t.Fatalf("did not track the total count of calls correctly. expected it to be 3, but it was %v", totalCallCount)
	}

	info := GetCallCountInfo()
	expectedInfo := map[string]int{}
	urlCallkey := "GET " + url
	url2Callkey := "POST " + url2
	expectedInfo[urlCallkey] = 2
	expectedInfo[url2Callkey] = 1

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
			RegisterResponderWithQuery("GET", testURLPath, query, NewStringResponder(200, body))

			for _, url := range test.URLs {
				req, err := http.NewRequest("GET", url, nil)
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
	resp := NewStringResponder(200, "hello world!")

	for _, test := range []struct {
		Query       interface{}
		PanicPrefix string
	}{
		{
			Query:       "%",
			PanicPrefix: "RegisterResponderWithQuery bad query string: ",
		},
		{
			Query:       1234,
			PanicPrefix: "RegisterResponderWithQuery bad query type int. Only url.Values, map[string]string and string are allowed",
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

			RegisterResponderWithQuery("GET", "foobar", test.Query, resp)
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

func TestCheckStackTracer(t *testing.T) {
	req, err := http.NewRequest("GET", "http://foo.bar/", nil)
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
	RegisterResponder("GET", url,
		NewStringResponder(200, "{}").
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
