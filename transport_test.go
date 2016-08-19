package httpmock

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

var testURL = "http://www.example.com/"

func TestMockTransport(t *testing.T) {
	type schema struct {
		Message string `xml:"message"`
	}

	Activate()
	defer DeactivateAndReset()

	url := "https://github.com/"
	body := &schema{"hello world"}

	responder, err := NewXMLResponder(200, body)
	if err != nil {
		t.Fatal(err)
	}

	RegisterStubRequest(NewStubRequest("GET", url, responder))

	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	checkBody := &schema{}
	if err := xml.NewDecoder(resp.Body).Decode(checkBody); err != nil {
		t.Fatal(err)
	}

	if checkBody.Message != body.Message {
		t.FailNow()
	}

	// the http client wraps our NoResponderFound error, so we just try and match
	// on text
	if _, err := http.Get(testURL); !strings.Contains(err.Error(),
		ErrNoResponderFound.Error()) {

		t.Fatal(err)
	}
}

func TestMockTransportCaseInsensitive(t *testing.T) {
	Activate()
	defer DeactivateAndReset()

	url := "https://github.com/"
	body := []byte("hello world")

	RegisterStubRequest(NewStubRequest("get", url, NewBytesResponder(200, body)))

	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != string(body) {
		t.FailNow()
	}

	// the http client wraps our NoResponderFound error, so we just try and match on text
	if _, err := http.Get(testURL); !strings.Contains(err.Error(),
		ErrNoResponderFound.Error()) {

		t.Fatal(err)
	}
}

func TestMockTransportAdvanced(t *testing.T) {
	type schema struct {
		Message string `json:"msg"`
	}

	Activate()
	defer DeactivateAndReset()

	url := "https://github.com/banana/"

	requestBody := `{"msg":"hello world"}`
	requestHeader := &http.Header{
		"X-ApiKey": []string{"api-key"},
	}
	responseBody := &schema{"ok"}

	responder, err := NewJSONResponder(200, responseBody)
	if err != nil {
		t.Fatalf("Unexpected error constructing request: %#v", err)
	}

	RegisterStubRequest(
		NewStubRequest(
			"POST",
			url,
			responder,
		).WithHeader(
			requestHeader,
		).WithBody(
			bytes.NewBufferString(requestBody),
		),
	)

	// should fail because missing stubbed header
	_, err = http.Post(url, "application/json", bytes.NewBufferString(requestBody))
	if err == nil {
		t.Fatalf("POST request should have failed due to missing headers")
	}

	client := &http.Client{}

	req, err := http.NewRequest("POST", url, bytes.NewBufferString(requestBody))
	if err != nil {
		t.Fatalf("Unexpected error constructing request: %#v", err)
	}

	req.Header.Add("X-ApiKey", "api-key")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Unexpected error when making request: %#v", err)
	}
	defer resp.Body.Close()

	checkBody := &schema{}
	if err := json.NewDecoder(resp.Body).Decode(checkBody); err != nil {
		t.Fatal(err)
	}

	if checkBody.Message != responseBody.Message {
		t.FailNow()
	}

	// verify that all stubs were called
	if err := AllStubsCalled(); err != nil {
		t.Errorf("Not all stubs were called: %s", err)
	}
}

func TestAllStubsCalled(t *testing.T) {
	Activate()
	defer DeactivateAndReset()

	// register two stubs
	RegisterStubRequest(NewStubRequest("GET", "http://github.com", NewStringResponder(200, "ok")))
	RegisterStubRequest(NewStubRequest("GET", "http://example.com", NewStringResponder(200, "ok")))

	// make a single request
	resp, err := http.Get("http://github.com")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	err = AllStubsCalled()
	if err == nil {
		t.Errorf("Expected error when not all stubs called")
	}

	if !strings.Contains(err.Error(), "http://example.com") {
		t.Errorf("Expected error message to contain uncalled stub, got: '%s'", err.Error())
	}
}

func TestMockTransportReset(t *testing.T) {
	DeactivateAndReset()

	if len(DefaultTransport.stubs) > 0 {
		t.Fatal("expected no responders at this point")
	}

	RegisterStubRequest(NewStubRequest("GET", testURL, nil))

	if len(DefaultTransport.stubs) != 1 {
		t.Fatal("expected one stubbed request")
	}

	Reset()

	if len(DefaultTransport.stubs) > 0 {
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

func TestMockTransportWithQuerystring(t *testing.T) {
	Activate()
	defer DeactivateAndReset()

	// register a responder with query parameters
	RegisterStubRequest(
		NewStubRequest(
			"GET",
			testURL+"?first=val&second=val",
			NewStringResponder(200, "hello world"),
		))

	// should error if no parameters passed
	if _, err := http.Get(testURL); err == nil {
		t.Fatal("expected to receive a connection error due to lack of responders")
	}

	// should error if if only one parameter passed
	if _, err := http.Get(testURL + "?first=val"); err == nil {
		t.Fatal("expected to receive a connection error due to lack of responders")
	}
	if _, err := http.Get(testURL + "?second=val"); err == nil {
		t.Fatal("expected to receive a connection error due to lack of responders")
	}

	// should error if more parameters passed
	if _, err := http.Get(testURL + "?first=val&second=val&third=val"); err == nil {
		t.Fatal("expected to receive a connection error due to lack of responders")
	}

	// should not error if both parameters are sent
	_, err := http.Get(testURL + "?first=val&second=val")
	if err != nil {
		t.Fatal("expected request to succeed")
	}

	_, err = http.Get(testURL + "?second=val&first=val")
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

	RegisterStubRequest(NewStubRequest("GET", testURL, NewStringResponder(200, body)))

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
