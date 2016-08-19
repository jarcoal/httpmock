package httpmock

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// ResponderFromResponse wraps an *http.Response in a Responder
func ResponderFromResponse(resp *http.Response) Responder {
	return func(req *http.Request) (*http.Response, error) {
		return resp, nil
	}
}

// NewStringResponse creates an *http.Response with a body based on the given string.  Also accepts
// an http status code.
func NewStringResponse(status int, body string) *http.Response {
	return &http.Response{
		Status:     strconv.Itoa(status),
		StatusCode: status,
		Body:       NewRespBodyFromString(body),
		Header:     http.Header{},
	}
}

// NewStringResponder creates a Responder from a given body (as a string) and status code.
func NewStringResponder(status int, body string) Responder {
	return ResponderFromResponse(NewStringResponse(status, body))
}

// NewBytesResponse creates an *http.Response with a body based on the given bytes.  Also accepts
// an http status code.
func NewBytesResponse(status int, body []byte) *http.Response {
	return &http.Response{
		Status:     strconv.Itoa(status),
		StatusCode: status,
		Body:       NewRespBodyFromBytes(body),
		Header:     http.Header{},
	}
}

// NewBytesResponder creates a Responder from a given body (as a byte slice) and status code.
func NewBytesResponder(status int, body []byte) Responder {
	return ResponderFromResponse(NewBytesResponse(status, body))
}

// NewJSONResponse creates an *http.Response with a body that is a json encoded representation of
// the given interface{}.  Also accepts an http status code.
func NewJSONResponse(status int, body interface{}) (*http.Response, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	response := NewBytesResponse(status, encoded)
	response.Header.Set("Content-Type", "application/json")
	return response, nil
}

// NewJSONResponder creates a Responder from a given body (as an interface{} that is encoded to
// json) and status code.
func NewJSONResponder(status int, body interface{}) (Responder, error) {
	resp, err := NewJSONResponse(status, body)
	if err != nil {
		return nil, err
	}
	return ResponderFromResponse(resp), nil
}

// NewXMLResponse creates an *http.Response with a body that is an xml encoded
// representation of the given interface{}.  Also accepts an http status code.
func NewXMLResponse(status int, body interface{}) (*http.Response, error) {
	encoded, err := xml.Marshal(body)
	if err != nil {
		return nil, err
	}
	response := NewBytesResponse(status, encoded)
	response.Header.Set("Content-Type", "application/xml")
	return response, nil
}

// NewXMLResponder creates a Responder from a given body (as an interface{}
// that is encoded to xml) and status code.
func NewXMLResponder(status int, body interface{}) (Responder, error) {
	resp, err := NewXMLResponse(status, body)
	if err != nil {
		return nil, err
	}
	return ResponderFromResponse(resp), nil
}

// NewRespBodyFromString creates an io.ReadCloser from a string that is
// suitable for use as an http response body.
func NewRespBodyFromString(body string) io.ReadCloser {
	return &dummyReadCloser{strings.NewReader(body)}
}

// NewRespBodyFromBytes creates an io.ReadCloser from a byte slice that is suitable for use as an
// http response body.
func NewRespBodyFromBytes(body []byte) io.ReadCloser {
	return &dummyReadCloser{bytes.NewReader(body)}
}

type dummyReadCloser struct {
	body io.ReadSeeker
}

func (d *dummyReadCloser) Read(p []byte) (n int, err error) {
	n, err = d.body.Read(p)
	if err == io.EOF {
		d.body.Seek(0, 0)
	}
	return n, err
}

func (d *dummyReadCloser) Close() error {
	return nil
}
