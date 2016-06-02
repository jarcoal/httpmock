package httpmock

import (
	"net/http"
)

// NewStubRequest is a constructor function that returns a StubRequest for the
// given method and url. We also supply a responder which actually generates
// the response should the stubbed request match the request.
func NewStubRequest(method, url string, responder Responder) *StubRequest {
	return &StubRequest{
		Method:    method,
		URL:       url,
		Responder: responder,
	}
}

// NewStubRequestWithHeaders is a constructor function that returns a
// StubRequest for the given method and url provided the request contains the
// supplied headers. We also supply a responder which actually generates the
// response should the stubbed request match the request.
func NewStubRequestWithHeaders(method, url string, header *http.Header, responder Responder) *StubRequest {
	return &StubRequest{
		Method:    method,
		URL:       url,
		Header:    header,
		Responder: responder,
	}
}

// StubRequest is used to capture data about a new stubbed request. It wraps up
// the Method and URL along with optional http.Header struct, holds the
// Responder used to generate a response, and also has a flag indicating
// whether or not this stubbed request has actually been called.
type StubRequest struct {
	Method    string
	URL       string
	Header    *http.Header
	Responder Responder
	Called    bool
}

// Matches is a test function that returns true if an incoming request is matched by this fetcher.
func (r *StubRequest) Matches(req *http.Request) bool {
	return false
}
