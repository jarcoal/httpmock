package httpmock

import (
	"net/http"
	"strings"
)

// NewStubRequest is a constructor function that returns a StubRequest for the
// given method and url. We also supply a responder which actually generates
// the response should the stubbed request match the request.
func NewStubRequest(method, url string, responder Responder) (*StubRequest, error) {
	return NewStubRequestWithHeaders(
		method,
		url,
		nil,
		responder)
}

// NewStubRequestWithHeaders is a constructor function that returns a
// StubRequest for the given method and url provided the request contains the
// supplied headers. We also supply a responder which actually generates the
// response should the stubbed request match the request.
func NewStubRequestWithHeaders(method, url string, header *http.Header, responder Responder) (*StubRequest, error) {
	normalized, err := normalizeURL(url)
	if err != nil {
		return nil, err
	}

	return &StubRequest{
		Method:    method,
		URL:       normalized,
		Header:    header,
		Responder: responder,
	}, nil
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

// Matches is a test function that returns true if an incoming request is
// matched by this fetcher. Should an incoming request URL cause an error when
// normalized, we return false.
func (r *StubRequest) Matches(req *http.Request) bool {
	methodMatch := strings.ToUpper(req.Method) == strings.ToUpper(r.Method)

	if !methodMatch {
		return methodMatch
	}

	normalized, err := normalizeURL(req.URL.String())
	if err != nil {
		return false
	}

	urlMatch := normalized == r.URL

	if !urlMatch {
		return urlMatch
	}

	headerMatch := true

	// only check headers if the stubbed request has set headers to some not nil
	// value
	if r.Header != nil {

	Loop:
		// for each header defined on the stub, iterate through all the values and
		// make sure they are present in the corresponding header on the request
		for header, stubValues := range map[string][]string(*r.Header) {
			// get the values for this header on the request
			reqValues := req.Header[header]

			for _, v := range stubValues {
				if !contains(reqValues, v) {
					headerMatch = false
					break Loop
				}
			}
		}
	}

	return headerMatch
}
