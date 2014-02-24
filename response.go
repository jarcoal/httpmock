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
func NewStringResponse(body string, status int) *http.Response {
	return &http.Response{
		Status:     strconv.Itoa(status),
		StatusCode: status,
		Body:       NewRespBodyFromString(body),
	}
}

// NewStringResponder creates a Responder from a given body (as a string) and status code.
func NewStringResponder(body string, status int) Responder {
	return ResponderFromResponse(NewStringResponse(body, status))
}

// NewBytesResponse creates an *http.Response with a body based on the given bytes.  Also accepts
// an http status code.
func NewBytesResponse(body []byte, status int) *http.Response {
	return &http.Response{
		Status:     strconv.Itoa(status),
		StatusCode: status,
		Body:       NewRespBodyFromBytes(body),
	}
}

// NewBytesResponder creates a Responder from a given body (as a byte slice) and status code.
func NewBytesResponder(body []byte, status int) Responder {
	return ResponderFromResponse(NewBytesResponse(body, status))
}

// NewJsonResponse creates an *http.Response with a body that is a json encoded representation of
// the given interface{}.  Also accepts an http status code.
func NewJsonResponse(body interface{}, status int) (*http.Response, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return NewBytesResponse(encoded, status), nil
}

// NewJsonResponder creates a Responder from a given body (as an interface{} that is encoded to
// json) and status code.
func NewJsonResponder(body interface{}, status int) (Responder, error) {
	resp, err := NewJsonResponse(body, status)
	if err != nil {
		return nil, err
	}
	return ResponderFromResponse(resp), nil
}

// NewXmlResponse creates an *http.Response with a body that is an xml encoded representation
// of the given interface{}.  Also accepts an http status code.
func NewXmlResponse(body interface{}, status int) (*http.Response, error) {
	encoded, err := xml.Marshal(body)
	if err != nil {
		return nil, err
	}
	return NewBytesResponse(encoded, status), nil
}

// NewXmlResponder creates a Responder from a given body (as an interface{} that is encoded to xml)
// and status code.
func NewXmlResponder(body interface{}, status int) (Responder, error) {
	resp, err := NewXmlResponse(body, status)
	if err != nil {
		return nil, err
	}
	return ResponderFromResponse(resp), nil
}

// NewRespBodyFromString creates an io.ReadCloser from a string that is suitable for use as an
// http response body.
func NewRespBodyFromString(body string) io.ReadCloser {
	return &dummyReadCloser{strings.NewReader(body)}
}

// NewRespBodyFromBytes creates an io.ReadCloser from a byte slice that is suitable for use as an
// http response body.
func NewRespBodyFromBytes(body []byte) io.ReadCloser {
	return &dummyReadCloser{bytes.NewReader(body)}
}

type dummyReadCloser struct {
	body io.Reader
}

func (d *dummyReadCloser) Read(p []byte) (n int, err error) {
	return d.body.Read(p)
}

func (d *dummyReadCloser) Close() error {
	return nil
}
