package httpmock

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

var testUrl = "http://www.example.com/"

func TestMockTransport(t *testing.T) {
	Activate(&http.DefaultTransport)
	defer Deactivate()

	url := "https://github.com/"
	body := "hello world"

	RegisterResponder("GET", url, NewStringResponder(200, body))

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

	if len(DefaultTransport.responders) > 0 {
		t.Fatal("expected no responders at this point")
	}

	RegisterResponder("GET", testUrl, nil)

	if len(DefaultTransport.responders) != 1 {
		t.Fatal("expected one responder")
	}

	Reset()

	if len(DefaultTransport.responders) > 0 {
		t.Fatal("expected no responders as they were just reset")
	}
}

func TestMockTransportNoResponder(t *testing.T) {
	Activate(&http.DefaultTransport)
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
	Activate(&http.DefaultTransport)
	defer DeactivateAndReset()

	// register the testUrl responder
	RegisterResponder("GET", testUrl, NewStringResponder(200, "hello world"))

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

type dummyTripper struct{}

func (d *dummyTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, nil
}

func TestMockTransportInitialTransport(t *testing.T) {
	DeactivateAndReset()

	tripper := &dummyTripper{}
	http.DefaultTransport = tripper

	Activate(&http.DefaultTransport)

	if http.DefaultTransport == tripper {
		t.Fatal("expected http.DefaultTransport to be a mock transport")
	}

	Deactivate()

	if http.DefaultTransport != tripper {
		t.Fatal("expected http.DefaultTransport to be dummy")
	}
}
