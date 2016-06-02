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

// StubNotCalled is a type implementing the error interface we return when a
// stub has not been called
type StubNotCalled struct {
	Stub *StubRequest
}

// Error ensures our StubNotCalled type implements the error interface
func (e *StubNotCalled) Error() string {
	return fmt.Sprintf("Stub not called: %#v", e.Stub)
}

// NewStubNotCalled returns a new StubNotCalled error for the given stub
func NewStubNotCalled(stub *StubRequest) *StubNotCalled {
	return &StubNotCalled{
		Stub: stub,
	}
}

// ConnectionFailure is a responder that returns a connection failure.  This is the default
// responder, and is called when no other matching responder is found.
func ConnectionFailure(*http.Request) (*http.Response, error) {
	return nil, NoResponderFound
}

// NewMockTransport creates a new *MockTransport with no stubbed requests.
func NewMockTransport() *MockTransport {
	return &MockTransport{
		make([]*StubRequest),
		nil,
	}
}

// MockTransport implements http.RoundTripper, which fulfills single http requests issued by
// an http.Client.  This implementation doesn't actually make the call, instead deferring to
// the registered list of stubbed requests.
type MockTransport struct {
	stubs       []StubRequest
	noResponder Responder
}

// RoundTrip receives HTTP requests and routes them to the appropriate responder.  It is required to
// implement the http.RoundTripper interface.  You will not interact with this directly, instead
// the *http.Client you are using will call it for you.
func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// try and get a responder that matches the given reque and URL
	stub := m.stubForRequest(req)

	// if we found a stub, call it's responder
	if stub != nil {
		// mark this stub as having been performed
		stub.Called = true

		return stub.Responder(req)
	}

	// we didn't find a responder, so fire the 'no responder' responder
	if m.noResponder == nil {
		return ConnectionFailure(req)
	}
	return m.noResponder(req)
}

// do nothing with timeout
func (m *MockTransport) CancelRequest(req *http.Request) {}

// stubForRequest returns the first matching stub for the incoming request
// object or nil if no stub claims to be a match
func (m *MockTransport) stubForRequest(req *http.Request) *StubRequest {
	// find the first stub that matches the request
	for _, stub := range m.stubs {
		if stub.Matches(req) {
			return stub
		}
	}

	return nil
}

// RegisterStubRequest adds a new responder, associated with a given stubbed
// request. When a request comes in that matches, the responder will be called
// and the response returned to the client.
func (m *MockTransport) RegisterStubRequest(stub *StubRequest) {
	m.stubs = append(m.stubs, stub)
}

// RegisterNoResponder is used to register a responder that will be called if
// no other responder is found.  The default is ConnectionFailure.
func (m *MockTransport) RegisterNoResponder(responder Responder) {
	m.noResponder = responder
}

// Reset removes all registered responders (including the no responder) from
// the MockTransport
func (m *MockTransport) Reset() {
	m.stubs = make([]*StubRequest)
	m.noResponder = nil
}

// AllStubsCalled returns nil if all of the currently registered stubs have
// been called; if some haven't been called, then it returns an error.
func (m *MockTransport) AllStubsCalled() error {
	for _, stub := range m.stubs {
		if stub.Called == false {
			return NewStubNotCalled(stub)
		}
	}

	return nil
}

// DefaultTransport is the default mock transport used by Activate, Deactivate,
// Reset, DeactivateAndReset, RegisterStubRequest, RegisterNoResponder and
// AllStubsCalled.
var DefaultTransport = NewMockTransport()

// InitialTransport is a cache of the original transport used so we can put it back
// when Deactivate is called.
var InitialTransport = http.DefaultTransport

// Used to handle custom http clients (i.e clients other than http.DefaultClient)
var oldTransport http.RoundTripper
var oldClient *http.Client

// Activate starts the mock environment.  This should be called before your tests run.  Under the
// hood this replaces the Transport on the http.DefaultClient with DefaultTransport.
//
// To enable mocks for a test, simply activate at the beginning of a test:
// 		func TestFetchArticles(t *testing.T) {
// 			httpmock.Activate()
// 			// all http requests will now be intercepted
// 		}
//
// If you want all of your tests in a package to be mocked, just call Activate from init():
// 		func init() {
// 			httpmock.Activate()
// 		}
func Activate() {
	if Disabled() {
		return
	}

	// make sure that if Activate is called multiple times it doesn't overwrite the InitialTransport
	// with a mock transport.
	if http.DefaultTransport != DefaultTransport {
		InitialTransport = http.DefaultTransport
	}

	http.DefaultTransport = DefaultTransport
}

// ActivateNonDefault starts the mock environment with a non-default http.Client.
// This emulates the Activate function, but allows for custom clients that do not use
// http.DefaultTransport
//
// To enable mocks for a test using a custom client, activate at the beginning of a test:
// 		client := &http.Client{Transport: &http.Transport{TLSHandshakeTimeout: 60 * time.Second}}
// 		httpmock.ActivateNonDefault(client)
func ActivateNonDefault(client *http.Client) {
	if Disabled() {
		return
	}

	// save the custom client & it's RoundTripper
	oldTransport = client.Transport
	oldClient = client
	client.Transport = DefaultTransport
}

// Deactivate shuts down the mock environment.  Any HTTP calls made after this will use a live
// transport.
//
// Usually you'll call it in a defer right after activating the mock environment:
// 		func TestFetchArticles(t *testing.T) {
// 			httpmock.Activate()
// 			defer httpmock.Deactivate()
//
// 			// when this test ends, the mock environment will close
// 		}
func Deactivate() {
	if Disabled() {
		return
	}
	http.DefaultTransport = InitialTransport

	// reset the custom client to use it's original RoundTripper
	if oldClient != nil {
		oldClient.Transport = oldTransport
	}
}

// Reset will remove any registered mocks and return the mock environment to it's initial state.
func Reset() {
	DefaultTransport.Reset()
}

// DeactivateAndReset is just a convenience method for calling Deactivate() and then Reset()
// Happy deferring!
func DeactivateAndReset() {
	Deactivate()
	Reset()
}

// RegisterStubRequest adds a mock that will catch requests to the given HTTP method and URL, then
// route them to the Responder which will generate a response to be returned to the client.
//
// Example:
// 		func TestFetchArticles(t *testing.T) {
// 			httpmock.Activate()
// 			defer httpmock.DeactivateAndReset()
//
// 			httpmock.RegisterStubRequest(&StubRequest{
// 				Method: "GET",
// 				URL: "http://example.com",
// 				Responder: httpmock.NewStringResponder("hello world", 200),
//			})
//
//			// requests to http://example.com/ will now return 'hello world'
// 		}
func RegisterStubRequest(request *StubRequest) {
	DefaultTransport.RegisterStubRequest(request)
}

// RegisterNoResponder adds a mock that will be called whenever a request for an unregistered URL
// is received.  The default behavior is to return a connection error.
//
// In some cases you may not want all URLs to be mocked, in which case you can do this:
// 		func TestFetchArticles(t *testing.T) {
// 			httpmock.Activate()
// 			defer httpmock.DeactivateAndReset()
//			httpmock.RegisterNoResponder(httpmock.InitialTransport.RoundTrip)
//
// 			// any requests that don't have a registered URL will be fetched normally
// 		}
func RegisterNoResponder(responder Responder) {
	DefaultTransport.RegisterNoResponder(responder)
}

// AllStubsCalled is a function intended to be used within your tests to
// verify that all registered stubs were actually invoked during the course of
// the test. Registering stubs but then not calling them is either a sign that
// something is wrong in your test, or a waste of time.  This function retuns
// an error unless all currently registered stubs were called.
//
// Example:
// 		func TestFetchArticles(t *testing.T) {
// 			httpmock.Activate()
// 			defer httpmock.DeactivateAndReset()
//
// 			httpmock.RegisterStubRequest(&StubRequest{
// 				Method: "GET",
// 				URL: "http://example.com",
// 				Responder: httpmock.NewStringResponder("hello world", 200),
//			})
//
//			// requests to http://example.com/ will now return 'hello world'
//			if err := httpmock.AllStubsCalled(); err != nil {
//				t.Errorf("Not all stubs were called: %s", err)
//			}
// 		}
func AllStubsCalled() error {
	return DefaultTransport.AllStubsCalled()
}
