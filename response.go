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
		res := new(http.Response)
		*res = *resp
		res.Request = req
		return res, nil
	}
}

// NewErrorResponder creates a Responder that returns an empty request and the
// given error. This can be used to e.g. imitate more deep http errors for the
// client.
func NewErrorResponder(err error) Responder {
	return func(req *http.Request) (*http.Response, error) {
		return nil, err
	}
}

// NewStringResponse creates an *http.Response with a body based on the given string.  Also accepts
// an http status code.
func NewStringResponse(status int, body string) *http.Response {
	return &http.Response{
		Status:        strconv.Itoa(status),
		StatusCode:    status,
		Body:          NewRespBodyFromString(body),
		Header:        http.Header{},
		ContentLength: -1,
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
		Status:        strconv.Itoa(status),
		StatusCode:    status,
		Body:          NewRespBodyFromBytes(body),
		Header:        http.Header{},
		ContentLength: -1,
	}
}

// NewBytesResponder creates a Responder from a given body (as a byte slice) and status code.
func NewBytesResponder(status int, body []byte) Responder {
	return ResponderFromResponse(NewBytesResponse(status, body))
}

// NewJsonResponse creates an *http.Response with a body that is a json encoded representation of
// the given interface{}.  Also accepts an http status code.
func NewJsonResponse(status int, body interface{}) (*http.Response, error) { // nolint: golint
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	response := NewBytesResponse(status, encoded)
	response.Header.Set("Content-Type", "application/json")
	return response, nil
}

// NewJsonResponder creates a Responder from a given body (as an interface{} that is encoded to
// json) and status code.
func NewJsonResponder(status int, body interface{}) (Responder, error) { // nolint: golint
	resp, err := NewJsonResponse(status, body)
	if err != nil {
		return nil, err
	}
	return ResponderFromResponse(resp), nil
}

// NewJsonResponderOrPanic is like NewJsonResponder but panics in case of error.
//
// It simplifies the call of RegisterResponder, avoiding the use of a
// temporary variable and an error check, and so can be used as
// NewStringResponder or NewBytesResponder in such context:
//   RegisterResponder(
//     "GET",
//     "/test/path",
//     NewJSONResponderOrPanic(200, &MyBody),
//   )
func NewJsonResponderOrPanic(status int, body interface{}) Responder { // nolint: golint
	responder, err := NewJsonResponder(status, body)
	if err != nil {
		panic(err)
	}
	return responder
}

// NewXmlResponse creates an *http.Response with a body that is an xml encoded representation
// of the given interface{}.  Also accepts an http status code.
func NewXmlResponse(status int, body interface{}) (*http.Response, error) { // nolint: golint
	encoded, err := xml.Marshal(body)
	if err != nil {
		return nil, err
	}
	response := NewBytesResponse(status, encoded)
	response.Header.Set("Content-Type", "application/xml")
	return response, nil
}

// NewXmlResponder creates a Responder from a given body (as an interface{} that is encoded to xml)
// and status code.
func NewXmlResponder(status int, body interface{}) (Responder, error) { // nolint: golint
	resp, err := NewXmlResponse(status, body)
	if err != nil {
		return nil, err
	}
	return ResponderFromResponse(resp), nil
}

// NewXmlResponderOrPanic is like NewXmlResponder but panics in case of error.
//
// It simplifies the call of RegisterResponder, avoiding the use of a
// temporary variable and an error check, and so can be used as
// NewStringResponder or NewBytesResponder in such context:
//   RegisterResponder(
//     "GET",
//     "/test/path",
//     NewXmlResponderOrPanic(200, &MyBody),
//   )
func NewXmlResponderOrPanic(status int, body interface{}) Responder { // nolint: golint
	responder, err := NewXmlResponder(status, body)
	if err != nil {
		panic(err)
	}
	return responder
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
	body io.ReadSeeker
}

func (d *dummyReadCloser) Read(p []byte) (n int, err error) {
	n, err = d.body.Read(p)
	if err == io.EOF {
		d.body.Seek(0, 0) // nolint: errcheck
	}
	return n, err
}

func (d *dummyReadCloser) Close() error {
	d.body.Seek(0, 0) // nolint: errcheck
	return nil
}
