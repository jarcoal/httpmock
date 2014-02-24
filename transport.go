package httpmock

import (
	"errors"
	"net/http"
	"strings"
)

// Responders are callbacks that receive and http request and return a mocked response.
type Responder func(*http.Request) (*http.Response, error)

// NoResponderFound is returned when no responders are found for a given HTTP method and URL.
var NoResponderFound = errors.New("no responder found")

// ConnectionFailure is a responder that returns a connection failure.  This is the default
// responder, and is called when no other matching responder is found.
func ConnectionFailure(*http.Request) (*http.Response, error) {
	return nil, NoResponderFound
}

func NewMockTransport() *MockTransport {
	return &MockTransport{true, make(map[string]Responder), nil}
}

// MockTransport implements http.RoundTripper, which fulfills single http requests issued by
// an http.Client.  This implementation doesn't actually make the call, instead deferring to
// the registered list of responders.
type MockTransport struct {
	IgnoreQueryString bool
	responders        map[string]Responder
	noResponder       Responder
}

// RoundTrip is required to implement http.MockTransport.  Instead of fulfilling the given request,
// the internal list of responders is consulted to handle the request.  If no responder is found
// an error is returned, which is the equivalent of a network error.
func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	url := req.URL.String()

	if m.IgnoreQueryString {
		url = strings.Split(url, "?")[0]
	}

	key := req.Method + " " + url

	// scan through the responders and find one that matches our key
	for k, r := range m.responders {
		if k != key {
			continue
		}
		return r(req)
	}

	// fire the 'no responder' responder
	if m.noResponder == nil {
		return ConnectionFailure(req)
	}
	return m.noResponder(req)
}

// RegisterResponder adds a new responder, associated with a given HTTP method and URL.  When a
// request comes in that matches, the responder will be called and the response returned to the client.
func (m *MockTransport) RegisterResponder(method, url string, responder Responder) {
	m.responders[method+" "+url] = responder
}

// RegisterNoResponder is used to register a responder that will be called if no other responder is
// found.  The default is `ConnectionFailure`
func (m *MockTransport) RegisterNoResponder(responder Responder) {
	m.noResponder = responder
}

// Reset removes all registered responders
func (m *MockTransport) Reset() {
	m.responders = make(map[string]Responder)
	m.noResponder = nil
}

// DefaultTransport allows users to easily and globally alter the default RoundTripper for
// all http requests.
var DefaultTransport = NewMockTransport()

// Activate replaces the `Transport` on the `http.DefaultClient` with our `DefaultTransport`.
func Activate() {
	if Disabled() {
		return
	}
	http.DefaultClient.Transport = DefaultTransport
}

// Deactivate replaces our `DefaultTransport` with the `http.DefaultTransport`.
func Deactivate() {
	if Disabled() {
		return
	}
	http.DefaultClient.Transport = http.DefaultTransport
}

// Reset resets the registered responders on the `DefaultTransport`
func Reset() {
	DefaultTransport.Reset()
}

// DeactivateAndReset is just a convenience method for calling `Deactivate()` and then `Reset()`
// Happy deferring!
func DeactivateAndReset() {
	Deactivate()
	Reset()
}

// RegisterResponder adds a responder to the `DefaultTransport` responder table.
func RegisterResponder(method, url string, responder Responder) {
	DefaultTransport.RegisterResponder(method, url, responder)
}

// RegisterNoResponder adds a 'no responder' to the `DefaultTransport`
func RegisterNoResponder(responder Responder) {
	DefaultTransport.RegisterNoResponder(responder)
}
