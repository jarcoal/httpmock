package httpmock

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
)

// Responder is a callback that receives and http request and returns
// a mocked response.
type Responder func(*http.Request) (*http.Response, error)

// NoResponderFound is returned when no responders are found for a given HTTP method and URL.
var NoResponderFound = errors.New("no responder found") // nolint: golint

// ConnectionFailure is a responder that returns a connection failure.  This is the default
// responder, and is called when no other matching responder is found.
func ConnectionFailure(*http.Request) (*http.Response, error) {
	return nil, NoResponderFound
}

// NewMockTransport creates a new *MockTransport with no responders.
func NewMockTransport() *MockTransport {
	return &MockTransport{
		responders:    make(map[string]Responder),
		callCountInfo: make(map[string]int),
	}
}

// MockTransport implements http.RoundTripper, which fulfills single http requests issued by
// an http.Client.  This implementation doesn't actually make the call, instead deferring to
// the registered list of responders.
type MockTransport struct {
	mu             sync.RWMutex
	responders     map[string]Responder
	noResponder    Responder
	callCountInfo  map[string]int
	totalCallCount int
}

// RoundTrip receives HTTP requests and routes them to the appropriate responder.  It is required to
// implement the http.RoundTripper interface.  You will not interact with this directly, instead
// the *http.Client you are using will call it for you.
func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	url := req.URL.String()

	method := req.Method
	if method == "" {
		// http.Request.Method is documented to default to GET:
		method = http.MethodGet
	}

	// try and get a responder that matches the method and URL with
	// query params untouched: http://z.tld/path?q...
	key := method + " " + url
	responder := m.responderForKey(key)

	// if we weren't able to find a responder, try with the URL *and*
	// sorted query params
	if responder == nil {
		query := sortedQuery(req.URL.Query())
		if query != "" {
			// Replace unsorted query params by sorted ones:
			//   http://z.tld/path?sorted_q...
			key = method + " " + strings.Replace(url, req.URL.RawQuery, query, 1)
			responder = m.responderForKey(key)
		}
	}

	// if we weren't able to find a responder, try without any query params
	if responder == nil {
		strippedURL := *req.URL
		strippedURL.RawQuery = ""
		strippedURL.Fragment = ""

		// go1.6 does not handle URL.ForceQuery, so in case it is set in go>1.6,
		// remove the "?" manually if present.
		surl := strings.TrimSuffix(strippedURL.String(), "?")

		hasQueryString := url != surl

		// if the URL contains a querystring then we strip off the
		// querystring and try again: http://z.tld/path
		if hasQueryString {
			key = method + " " + surl
			responder = m.responderForKey(key)
		}

		// if we weren't able to find a responder for the full URL, try with
		// the path part only
		if responder == nil {
			keyPathAlone := method + " " + req.URL.Path

			// First with unsorted querystring: /path?q...
			if hasQueryString {
				key = keyPathAlone + strings.TrimPrefix(url, surl) // concat after-path part
				responder = m.responderForKey(key)

				// Then with sorted querystring: /path?sorted_q...
				if responder == nil {
					key = keyPathAlone + "?" + sortedQuery(req.URL.Query())
					if req.URL.Fragment != "" {
						key += "#" + req.URL.Fragment
					}
					responder = m.responderForKey(key)
				}
			}

			// Then using path alone: /path
			if responder == nil {
				key = keyPathAlone
				responder = m.responderForKey(key)
			}
		}
	}

	m.mu.Lock()
	// if we found a responder, call it
	if responder != nil {
		m.callCountInfo[key]++
		m.totalCallCount++
	} else {
		// we didn't find a responder, so fire the 'no responder' responder
		if m.noResponder != nil {
			m.callCountInfo["NO_RESPONDER"]++
			m.totalCallCount++
			responder = m.noResponder
		}
	}
	m.mu.Unlock()

	if responder == nil {
		return ConnectionFailure(req)
	}
	return runCancelable(responder, req)
}

func runCancelable(responder Responder, req *http.Request) (*http.Response, error) {
	// TODO: replace req.Cancel by ctx
	if req.Cancel == nil { // nolint: staticcheck
		return responder(req)
	}

	// Set up a goroutine that translates a close(req.Cancel) into a
	// "request canceled" error, and another one that runs the
	// responder. Then race them: first to the result channel wins.

	type result struct {
		response *http.Response
		err      error
	}
	resultch := make(chan result, 1)
	done := make(chan struct{}, 1)

	go func() {
		select {
		// TODO: req.Cancel replace by ctx
		case <-req.Cancel: // nolint: staticcheck
			resultch <- result{
				response: nil,
				err:      errors.New("request canceled"),
			}
		case <-done:
		}
	}()

	go func() {
		defer func() {
			if err := recover(); err != nil {
				resultch <- result{
					response: nil,
					err:      fmt.Errorf("panic in responder: got %q", err),
				}
			}
		}()

		response, err := responder(req)
		resultch <- result{
			response: response,
			err:      err,
		}
	}()

	r := <-resultch

	// if a close(req.Cancel) is never coming,
	// we'll need to unblock the first goroutine.
	done <- struct{}{}

	return r.response, r.err
}

// CancelRequest does nothing with timeout.
func (m *MockTransport) CancelRequest(req *http.Request) {}

// responderForKey returns a responder for a given key.
func (m *MockTransport) responderForKey(key string) Responder {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.responders[key]
}

// RegisterResponder adds a new responder, associated with a given
// HTTP method and URL (or path).
//
// When a request comes in that matches, the responder will be called
// and the response returned to the client.
//
// If url contains query parameters, their order matters.
func (m *MockTransport) RegisterResponder(method, url string, responder Responder) {
	key := method + " " + url

	m.mu.Lock()
	m.responders[key] = responder
	m.callCountInfo[key] = 0
	m.mu.Unlock()
}

// RegisterResponderWithQuery is same as RegisterResponder, but it
// doesn't depend on query items order.
//
// query type can be:
//   url.Values
//   map[string]string
//   string, a query string like "a=12&a=13&b=z&c" (see net/url.ParseQuery function)
//
// If the query type is not recognized or the string cannot be parsed
// using net/url.ParseQuery, a panic() occurs.
func (m *MockTransport) RegisterResponderWithQuery(method, path string, query interface{}, responder Responder) {
	var mapQuery url.Values
	switch q := query.(type) {
	case url.Values:
		mapQuery = q

	case map[string]string:
		mapQuery = make(url.Values, len(q))
		for key, e := range q {
			mapQuery[key] = []string{e}
		}

	case string:
		var err error
		mapQuery, err = url.ParseQuery(q)
		if err != nil {
			panic("RegisterResponderWithQuery bad query string: " + err.Error())
		}

	default:
		panic(fmt.Sprintf("RegisterResponderWithQuery bad query type %T. Only url.Values, map[string]string and string are allowed", query))
	}

	if queryString := sortedQuery(mapQuery); queryString != "" {
		path += "?" + queryString
	}
	m.RegisterResponder(method, path, responder)
}

func sortedQuery(m url.Values) string {
	if len(m) == 0 {
		return ""
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b bytes.Buffer
	var values []string // nolint: prealloc

	for _, k := range keys {
		// Do not alter the passed url.Values
		values = append(values, m[k]...)
		sort.Strings(values)

		k = url.QueryEscape(k)

		for _, v := range values {
			if b.Len() > 0 {
				b.WriteByte('&')
			}
			fmt.Fprintf(&b, "%v=%v", k, url.QueryEscape(v))
		}

		values = values[:0]
	}

	return b.String()
}

// RegisterNoResponder is used to register a responder that will be called if no other responder is
// found.  The default is ConnectionFailure.
func (m *MockTransport) RegisterNoResponder(responder Responder) {
	m.mu.Lock()
	m.noResponder = responder
	m.mu.Unlock()
}

// Reset removes all registered responders (including the no responder) from the MockTransport
func (m *MockTransport) Reset() {
	m.mu.Lock()
	m.responders = make(map[string]Responder)
	m.noResponder = nil
	m.callCountInfo = make(map[string]int)
	m.totalCallCount = 0
	m.mu.Unlock()
}

// GetCallCountInfo returns callCountInfo
func (m *MockTransport) GetCallCountInfo() map[string]int {
	res := map[string]int{}
	m.mu.RLock()
	for k, v := range m.callCountInfo {
		res[k] = v
	}
	m.mu.RUnlock()
	return res
}

// GetTotalCallCount returns the totalCallCount
func (m *MockTransport) GetTotalCallCount() int {
	m.mu.RLock()
	count := m.totalCallCount
	m.mu.RUnlock()
	return count
}

// DefaultTransport is the default mock transport used by Activate, Deactivate, Reset,
// DeactivateAndReset, RegisterResponder, and RegisterNoResponder.
var DefaultTransport = NewMockTransport()

// InitialTransport is a cache of the original transport used so we can put it back
// when Deactivate is called.
var InitialTransport = http.DefaultTransport

// Used to handle custom http clients (i.e clients other than http.DefaultClient)
var oldClients = map[*http.Client]http.RoundTripper{}

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
	if _, ok := oldClients[client]; !ok {
		oldClients[client] = client.Transport
	}
	client.Transport = DefaultTransport
}

// GetCallCountInfo gets the info on all the calls httpmock has taken since it was activated or
// reset. The info is returned as a map of the calling keys with the number of calls made to them
// as their value. The key is the method, a space, and the url all concatenated together.
func GetCallCountInfo() map[string]int {
	return DefaultTransport.GetCallCountInfo()
}

// GetTotalCallCount gets the total number of calls httpmock has taken since it was activated or
// reset.
func GetTotalCallCount() int {
	return DefaultTransport.GetTotalCallCount()
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

	// reset the custom clients to use their original RoundTripper
	for oldClient, oldTransport := range oldClients {
		oldClient.Transport = oldTransport
		delete(oldClients, oldClient)
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

// RegisterResponder adds a mock that will catch requests to the given HTTP method and URL (or path), then
// route them to the Responder which will generate a response to be returned to the client.
//
// Example:
// 		func TestFetchArticles(t *testing.T) {
// 			httpmock.Activate()
// 			httpmock.DeactivateAndReset()
//
// 			httpmock.RegisterResponder("GET", "http://example.com/",
// 				httpmock.NewStringResponder(200, "hello world"))
//
// 			httpmock.RegisterResponder("GET", "/path/only",
// 				httpmock.NewStringResponder("any host hello world", 200))
//
//			// requests to http://example.com/ will now return 'hello world' and
//			// requests to any host with path /path/only will return 'any host hello world'
// 		}
func RegisterResponder(method, url string, responder Responder) {
	DefaultTransport.RegisterResponder(method, url, responder)
}

// RegisterResponderWithQuery it is same as RegisterResponder, but
// doesn't depends on query items order.
//
// query type can be:
//   url.Values
//   map[string]string
//   string, a query string like "a=12&a=13&b=z&c" (see net/url.ParseQuery function)
//
// If the query type is not recognized or the string cannot be parsed
// using net/url.ParseQuery, a panic() occurs.
//
// Example using a net/url.Values:
// 		func TestFetchArticles(t *testing.T) {
// 			httpmock.Activate()
// 			httpmock.DeactivateAndReset()
//
// 			expectedQuery := net.Values{
//				"a": []string{"3", "1", "8"},
//				"b": []string{"4", "2"},
//			}
// 			httpmock.RegisterResponderWithQueryValues("GET", "http://example.com/", expectedQuery,
// 				httpmock.NewStringResponder("hello world", 200))
//
//			// requests to http://example.com?a=1&a=3&a=8&b=2&b=4
//			//      and to http://example.com?b=4&a=2&b=2&a=8&a=1
//			// will now return 'hello world'
// 		}
//
// or using a map[string]string:
// 		func TestFetchArticles(t *testing.T) {
// 			httpmock.Activate()
// 			httpmock.DeactivateAndReset()
//
// 			expectedQuery := map[string]string{
//				"a": "1",
//				"b": "2"
//			}
// 			httpmock.RegisterResponderWithQuery("GET", "http://example.com/", expectedQuery,
// 				httpmock.NewStringResponder("hello world", 200))
//
//			// requests to http://example.com?a=1&b=2 and http://example.com?b=2&a=1 will now return 'hello world'
// 		}
//
// or using a query string:
// 		func TestFetchArticles(t *testing.T) {
// 			httpmock.Activate()
// 			httpmock.DeactivateAndReset()
//
// 			expectedQuery := "a=3&b=4&b=2&a=1&a=8"
// 			httpmock.RegisterResponderWithQueryValues("GET", "http://example.com/", expectedQuery,
// 				httpmock.NewStringResponder("hello world", 200))
//
//			// requests to http://example.com?a=1&a=3&a=8&b=2&b=4
//			//      and to http://example.com?b=4&a=2&b=2&a=8&a=1
//			// will now return 'hello world'
// 		}
func RegisterResponderWithQuery(method, path string, query interface{}, responder Responder) {
	DefaultTransport.RegisterResponderWithQuery(method, path, query, responder)
}

// RegisterResponderWithQueryValues it is same as RegisterResponder, but doesn't depends on query objects order.
//
// Example:

// RegisterNoResponder adds a mock that will be called whenever a request for an unregistered URL
// is received.  The default behavior is to return a connection error.
//
// In some cases you may not want all URLs to be mocked, in which case you can do this:
// 		func TestFetchArticles(t *testing.T) {
// 			httpmock.Activate()
// 			httpmock.DeactivateAndReset()
//			httpmock.RegisterNoResponder(httpmock.InitialTransport.RoundTrip)
//
// 			// any requests that don't have a registered URL will be fetched normally
// 		}
func RegisterNoResponder(responder Responder) {
	DefaultTransport.RegisterNoResponder(responder)
}
