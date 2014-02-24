package httpmock

import (
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"testing"
)

func TestNewStringResponse(t *testing.T) {
	body := "hello world"
	status := 200
	response := NewStringResponse(body, status)

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
	response := NewBytesResponse(body, status)

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

	response, err := NewJsonResponse(body, status)
	if err != nil {
		t.Fatal(err)
	}

	if response.StatusCode != status {
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

	response, err := NewXmlResponse(body, status)
	if err != nil {
		t.Fatal(err)
	}

	if response.StatusCode != status {
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
