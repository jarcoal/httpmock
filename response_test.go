package httpmock

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

func TestResponderFromResponse(t *testing.T) {
	responder := ResponderFromResponse(NewStringResponse(200, "hello world"))

	req, err := http.NewRequest(http.MethodGet, testURL, nil)
	if err != nil {
		t.Fatal("Error creating request")
	}
	response1, err := responder(req)
	if err != nil {
		t.Error("Error should be nil")
	}

	testURLWithQuery := testURL + "?a=1"
	req, err = http.NewRequest(http.MethodGet, testURLWithQuery, nil)
	if err != nil {
		t.Fatal("Error creating request")
	}
	response2, err := responder(req)
	if err != nil {
		t.Error("Error should be nil")
	}

	// Body should be the same for both responses
	assertBody(t, response1, "hello world")
	assertBody(t, response2, "hello world")

	// Request should be non-nil and different for each response
	if response1.Request != nil && response2.Request != nil {
		if response1.Request.URL.String() != testURL {
			t.Errorf("Expected request url %s, got: %s", testURL, response1.Request.URL.String())
		}
		if response2.Request.URL.String() != testURLWithQuery {
			t.Errorf("Expected request url %s, got: %s", testURLWithQuery, response2.Request.URL.String())
		}
	} else {
		t.Error("response.Request should not be nil")
	}
}

func TestNewNotFoundResponder(t *testing.T) {
	var mesg string
	responder := NewNotFoundResponder(func(args ...interface{}) {
		mesg = fmt.Sprint(args[0])
	})

	req, err := http.NewRequest("GET", "http://foo.bar/path", nil)
	if err != nil {
		t.Fatal("Error creating request")
	}

	const title = "Responder not found for GET http://foo.bar/path"

	resp, err := responder(req)
	if resp != nil {
		t.Error("resp should be nil")
	}
	if err == nil {
		t.Error("err should be not nil")
	} else if err.Error() != title {
		t.Errorf(`err mismatch, got: "%s", expected: "%s"`,
			err.Error(),
			"Responder not found for: GET http://foo.bar/path")
	}

	if !strings.HasPrefix(mesg, title+"\nCalled from ") {
		t.Error(`mesg should begin with "` + title + `\nCalled from ", but it is: "` + mesg + `"`)
	}
	if strings.HasSuffix(mesg, "\n") {
		t.Error(`mesg should not end with \n, but it is: "` + mesg + `"`)
	}

	// nil fn
	responder = NewNotFoundResponder(nil)

	resp, err = responder(req)
	if resp != nil {
		t.Error("resp should be nil")
	}
	if err == nil {
		t.Error("err should be not nil")
	} else if err.Error() != title {
		t.Errorf(`err mismatch, got: "%s", expected: "%s"`,
			err.Error(),
			"Responder not found for: GET http://foo.bar/path")
	}
}

func TestNewStringResponse(t *testing.T) {
	body := "hello world"
	status := 200
	response := NewStringResponse(status, body)

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != body {
		t.FailNow()
	}

	if response.StatusCode != status {
		t.FailNow()
	}
}

func TestNewBytesResponse(t *testing.T) {
	body := []byte("hello world")
	status := 200
	response := NewBytesResponse(status, body)

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != string(body) {
		t.FailNow()
	}

	if response.StatusCode != status {
		t.FailNow()
	}
}

func TestNewJsonResponse(t *testing.T) {
	type schema struct {
		Hello string `json:"hello"`
	}

	body := &schema{"world"}
	status := 200

	response, err := NewJsonResponse(status, body)
	if err != nil {
		t.Fatal(err)
	}

	if response.StatusCode != status {
		t.FailNow()
	}

	if response.Header.Get("Content-Type") != "application/json" {
		t.FailNow()
	}

	checkBody := &schema{}
	if err := json.NewDecoder(response.Body).Decode(checkBody); err != nil {
		t.Fatal(err)
	}

	if checkBody.Hello != body.Hello {
		t.FailNow()
	}
}

func TestNewXmlResponse(t *testing.T) {
	type schema struct {
		Hello string `xml:"hello"`
	}

	body := &schema{"world"}
	status := 200

	response, err := NewXmlResponse(status, body)
	if err != nil {
		t.Fatal(err)
	}

	if response.StatusCode != status {
		t.FailNow()
	}

	if response.Header.Get("Content-Type") != "application/xml" {
		t.FailNow()
	}

	checkBody := &schema{}
	if err := xml.NewDecoder(response.Body).Decode(checkBody); err != nil {
		t.Fatal(err)
	}

	if checkBody.Hello != body.Hello {
		t.FailNow()
	}
}

func TestNewErrorResponder(t *testing.T) {
	// From go1.13, a stack frame is stored into errors issued by errors.New()
	origError := errors.New("oh no")
	responder := NewErrorResponder(origError)
	req, err := http.NewRequest(http.MethodGet, testURL, nil)
	if err != nil {
		t.Fatal("Error creating request")
	}
	response, err := responder(req)
	if response != nil {
		t.Error("Response should be nil")
	}
	if err != origError {
		t.Errorf("Expected error %#v, got: %#v", origError, err)
	}
}

func TestRewindResponse(t *testing.T) {
	body := []byte("hello world")
	status := 200
	responses := []*http.Response{
		NewBytesResponse(status, body),
		NewStringResponse(status, string(body)),
	}

	for _, response := range responses {

		data, err := ioutil.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}

		if string(data) != string(body) {
			t.FailNow()
		}

		if response.StatusCode != status {
			t.FailNow()
		}

		data, err = ioutil.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}

		if string(data) != string(body) {
			t.FailNow()
		}

		if response.StatusCode != status {
			t.FailNow()
		}
	}
}
