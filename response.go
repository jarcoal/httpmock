package httpmock

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
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

// NewNotFoundResponder creates a Responder typically used in
// conjunction with RegisterNoResponder() function and testing
// package, to be proactive when a Responder is not found. fn is
// called with a unique string parameter containing the name of the
// missing route and the stack trace to localize the origin of the
// call. If fn returns (= if it does not panic), the responder returns
// an error of the form: "Responder not found for GET http://foo.bar/path".
// Note that fn can be nil.
//
// It is useful when writing tests to ensure that all routes have been
// mocked.
//
// Example of use:
//   import "testing"
//   ...
//   func TestMyApp(t *testing.T) {
//   	...
//   	// Calls testing.Fatal with the name of Responder-less route and
//   	// the stack trace of the call.
//   	httpmock.RegisterNoResponder(httpmock.NewNotFoundResponder(t.Fatal))
//
// Will abort the current test and print something like:
//   response:69: Responder not found for: GET http://foo.bar/path
//       Called from goroutine 20 [running]:
//         github.com/jarcoal/httpmock.NewNotFoundResponder.func1(0xc00011f000, 0x0, 0x42dfb1, 0x77ece8)
//           /go/src/github.com/jarcoal/httpmock/response.go:67 +0x1c1
//         github.com/jarcoal/httpmock.runCancelable(0xc00004bfc0, 0xc00011f000, 0x7692f8, 0xc, 0xc0001208b0)
//           /go/src/github.com/jarcoal/httpmock/transport.go:146 +0x7e
//         github.com/jarcoal/httpmock.(*MockTransport).RoundTrip(0xc00005c980, 0xc00011f000, 0xc00005c980, 0x0, 0x0)
//           /go/src/github.com/jarcoal/httpmock/transport.go:140 +0x19d
//         net/http.send(0xc00011f000, 0x7d3440, 0xc00005c980, 0x0, 0x0, 0x0, 0xc000010400, 0xc000047bd8, 0x1, 0x0)
//           /usr/local/go/src/net/http/client.go:250 +0x461
//         net/http.(*Client).send(0x9f6e20, 0xc00011f000, 0x0, 0x0, 0x0, 0xc000010400, 0x0, 0x1, 0x9f7ac0)
//         	 /usr/local/go/src/net/http/client.go:174 +0xfb
//         net/http.(*Client).do(0x9f6e20, 0xc00011f000, 0x0, 0x0, 0x0)
//         	 /usr/local/go/src/net/http/client.go:641 +0x279
//         net/http.(*Client).Do(...)
//         	 /usr/local/go/src/net/http/client.go:509
//         net/http.(*Client).Get(0x9f6e20, 0xc00001e420, 0x23, 0xc00012c000, 0xb, 0x600)
//         	 /usr/local/go/src/net/http/client.go:398 +0x9e
//         net/http.Get(...)
//         	 /usr/local/go/src/net/http/client.go:370
//         foo.bar/foobar/foobar.TestMyApp(0xc00011e000)
//         	 /go/src/foo.bar/foobar/foobar/my_app_test.go:272 +0xdbb
//         testing.tRunner(0xc00011e000, 0x77e3a8)
//         	 /usr/local/go/src/testing/testing.go:865 +0xc0
//         created by testing.(*T).Run
//         	 /usr/local/go/src/testing/testing.go:916 +0x35a
func NewNotFoundResponder(fn func(...interface{})) Responder {
	return func(req *http.Request) (*http.Response, error) {
		mesg := fmt.Sprintf("Responder not found for %s %s", req.Method, req.URL)
		if fn != nil {
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			buf = buf[:n]
			fn(mesg + "\nCalled from " +
				strings.Replace(strings.TrimSuffix(string(buf), "\n"), "\n", "\n  ", -1))
		}
		return nil, errors.New(mesg)
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
