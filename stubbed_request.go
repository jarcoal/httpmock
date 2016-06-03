package httpmock

import (
	"io"
	"io/ioutil"
	"net/http"
	"strings"
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

// StubRequest is used to capture data about a new stubbed request. It wraps up
// the Method and URL along with optional http.Header struct, holds the
// Responder used to generate a response, and also has a flag indicating
// whether or not this stubbed request has actually been called.
type StubRequest struct {
	Method    string
	URL       string
	Header    *http.Header
	Body      io.Reader
	Responder Responder
	Called    bool
}

// WithHeaders is a function used to add http headers onto a stubbed request.
func (r *StubRequest) WithHeaders(header *http.Header) *StubRequest {
	r.Header = header
	return r
}

// WithBody is a function used to add a body to a stubbed request
func (r *StubRequest) WithBody(body io.Reader) *StubRequest {
	r.Body = body
	return r
}

// Matches is a test function that returns true if an incoming request is
// matched by this fetcher. Should an incoming request URL cause an error when
// normalized, we return false.
func (r *StubRequest) Matches(req *http.Request) bool {
	methodMatch := strings.ToUpper(req.Method) == strings.ToUpper(r.Method)

	if !methodMatch {
		return methodMatch
	}

	normalizedURL, err := normalizeURL(r.URL)
	if err != nil {
		return false
	}

	normalizedReqUrl, err := normalizeURL(req.URL.String())
	if err != nil {
		return false
	}

	urlMatch := normalizedURL == normalizedReqUrl

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

	if !headerMatch {
		return headerMatch
	}

	bodyMatch := true

	// if our stub includes a body, then it should equal the actual request body
	// to match
	if r.Body != nil {
		stubBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return false
		}

		requestBody, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return false
		}

		bodyMatch = string(stubBody) == string(requestBody)
	}

	return bodyMatch
}
