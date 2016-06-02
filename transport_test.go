package httpmock

import (
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

var testUrl = "http://www.example.com/"

func TestMockTransport(t *testing.T) {
	Activate()
	defer Deactivate()

	url := "https://github.com/"
	body := "hello world"

	RegisterStubRequest(&StubRequest{
		Method:    "GET",
		URL:       url,
		Responder: NewStringResponder(200, body),
	})

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
	if _, err := http.Get(testUrl); !strings.Contains(err.Error(),
		NoResponderFound.Error()) {

		t.Fatal(err)
	}
}

func TestMockTransportCaseInsensitive(t *testing.T) {
	Activate()
	defer Deactivate()

	url := "https://github.com/"
	body := "hello world"

	RegisterStubRequest(&StubRequest{
		Method:    "get",
		URL:       url,
		Responder: NewStringResponder(200, body),
	})

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
	if _, err := http.Get(testUrl); !strings.Contains(err.Error(),
		NoResponderFound.Error()) {

		t.Fatal(err)
	}
}

func TestMockTransportReset(t *testing.T) {
	DeactivateAndReset()

	if len(DefaultTransport.requests) > 0 {
		t.Fatal("expected no responders at this point")
	}

	RegisterStubRequest(&StubRequest{
		Method:    "GET",
		URL:       testUrl,
		Responder: nil,
	})

	if len(DefaultTransport.requests) != 1 {
		t.Fatal("expected one stubbed request")
	}

	Reset()

	if len(DefaultTransport.requests) > 0 {
		t.Fatal("expected no stubbed requests as they were just reset")
	}
}

func TestMockTransportNoResponder(t *testing.T) {
	Activate()
	defer DeactivateAndReset()

	Reset()

	if DefaultTransport.noResponder != nil {
		t.Fatal("expected noResponder to be nil")
	}

	if _, err := http.Get(testUrl); err == nil {
		t.Fatal("expected to receive a connection error due to lack of responders")
	}

	RegisterNoResponder(NewStringResponder(200, "hello world"))

	resp, err := http.Get(testUrl)
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

	// register the testUrl responder
	RegisterStubRequest(&StubRequest{
		Method:    "GET",
		URL:       testUrl,
		Responder: NewStringResponder(200, "hello world"),
	})

	// make a request for the testUrl with a querystring
	resp, err := http.Get(testUrl + "?hello=world")
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

func TestMockTransportWithQuerystring(t *testing.T) {
	Activate()
	defer DeactivateAndReset()

	// register a responder with query parameters
	RegisterStubRequest(&StubRequest{
		Method:    "GET",
		URL:       testUrl + "?first=val&second=val",
		Responder: NewStringResponder(200, "hello world"),
	})

	// should error if no parameters passed
	if _, err := http.Get(testUrl); err == nil {
		t.Fatal("expected to receive a connection error due to lack of responders")
	}

	// should error if if only one parameter passed
	if _, err := http.Get(testUrl + "?first=val"); err == nil {
		t.Fatal("expected to receive a connection error due to lack of responders")
	}
	if _, err := http.Get(testUrl + "?second=val"); err == nil {
		t.Fatal("expected to receive a connection error due to lack of responders")
	}

	// should error if more parameters passed
	if _, err := http.Get(testUrl + "?first=val&second=val&third=val"); err == nil {
		t.Fatal("expected to receive a connection error due to lack of responders")
	}

	// should not error if both parameters are sent
	_, err := http.Get(testUrl + "?first=val&second=val")
	if err != nil {
		t.Fatal("expected request to succeed")
	}

	_, err = http.Get(testUrl + "?second=val&first=val")
	if err != nil {
		t.Fatal("expected request to succeed")
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

	RegisterStubRequest(&StubRequest{
		Method:    "GET",
		URL:       testUrl,
		Responder: NewStringResponder(200, body),
	})

	req, err := http.NewRequest("GET", testUrl, nil)
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
