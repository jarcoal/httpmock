package httpmock_test

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maxatome/go-testdeep/td"

	. "github.com/jarcoal/httpmock"
	"github.com/jarcoal/httpmock/internal"
)

func TestResponderFromResponse(t *testing.T) {
	assert, require := td.AssertRequire(t)

	responder := ResponderFromResponse(NewStringResponse(200, "hello world"))

	req, err := http.NewRequest(http.MethodGet, testURL, nil)
	require.CmpNoError(err)

	response1, err := responder(req)
	require.CmpNoError(err)

	testURLWithQuery := testURL + "?a=1"
	req, err = http.NewRequest(http.MethodGet, testURLWithQuery, nil)
	require.CmpNoError(err)

	response2, err := responder(req)
	require.CmpNoError(err)

	// Body should be the same for both responses
	assertBody(t, response1, "hello world")
	assertBody(t, response2, "hello world")

	// Request should be non-nil and different for each response
	require.NotNil(response1.Request)
	assert.String(response1.Request.URL, testURL)

	require.NotNil(response2.Request)
	assert.String(response2.Request.URL, testURLWithQuery)
}

func TestResponderFromResponses(t *testing.T) {
	assert, require := td.AssertRequire(t)

	jsonResponse, err := NewJsonResponse(200, map[string]string{"test": "toto"})
	require.CmpNoError(err)

	responder := ResponderFromMultipleResponses(
		[]*http.Response{
			jsonResponse,
			NewStringResponse(200, "hello world"),
		},
	)

	req, err := http.NewRequest(http.MethodGet, testURL, nil)
	require.CmpNoError(err)

	response1, err := responder(req)
	require.CmpNoError(err)

	testURLWithQuery := testURL + "?a=1"
	req, err = http.NewRequest(http.MethodGet, testURLWithQuery, nil)
	require.CmpNoError(err)

	response2, err := responder(req)
	require.CmpNoError(err)

	// Body should be the same for both responses
	assertBody(t, response1, `{"test":"toto"}`)
	assertBody(t, response2, "hello world")

	// Request should be non-nil and different for each response
	require.NotNil(response1.Request)
	assert.String(response1.Request.URL, testURL)

	require.NotNil(response2.Request)
	assert.String(response2.Request.URL, testURLWithQuery)

	// ensure we can't call the responder more than the number of responses it embeds
	_, err = responder(req)
	assert.String(err, "not enough responses provided: responder called 3 time(s) but 2 response(s) provided")

	// fn usage
	responder = ResponderFromMultipleResponses([]*http.Response{}, func(args ...interface{}) {})
	_, err = responder(req)
	assert.String(err, "not enough responses provided: responder called 1 time(s) but 0 response(s) provided")
	if assert.Isa(err, internal.StackTracer{}) {
		assert.NotNil(err.(internal.StackTracer).CustomFn)
	}
}

func TestNewNotFoundResponder(t *testing.T) {
	assert, require := td.AssertRequire(t)

	responder := NewNotFoundResponder(func(args ...interface{}) {})

	req, err := http.NewRequest("GET", "http://foo.bar/path", nil)
	require.CmpNoError(err)

	const title = "Responder not found for GET http://foo.bar/path"

	resp, err := responder(req)
	assert.Nil(resp)
	assert.String(err, title)
	if assert.Isa(err, internal.StackTracer{}) {
		assert.NotNil(err.(internal.StackTracer).CustomFn)
	}

	// nil fn
	responder = NewNotFoundResponder(nil)

	resp, err = responder(req)
	assert.Nil(resp)
	assert.String(err, title)
	if assert.Isa(err, internal.StackTracer{}) {
		assert.Nil(err.(internal.StackTracer).CustomFn)
	}
}

func TestNewStringResponse(t *testing.T) {
	assert, require := td.AssertRequire(t)

	const (
		body   = "hello world"
		status = 200
	)
	response := NewStringResponse(status, body)

	data, err := ioutil.ReadAll(response.Body)
	require.CmpNoError(err)

	assert.String(data, body)
	assert.Cmp(response.StatusCode, status)
}

func TestNewBytesResponse(t *testing.T) {
	assert, require := td.AssertRequire(t)

	const (
		body   = "hello world"
		status = 200
	)
	response := NewBytesResponse(status, []byte(body))

	data, err := ioutil.ReadAll(response.Body)
	require.CmpNoError(err)

	assert.String(data, body)
	assert.Cmp(response.StatusCode, status)
}

func TestNewJsonResponse(t *testing.T) {
	assert := td.Assert(t)

	type schema struct {
		Hello string `json:"hello"`
	}

	dir, cleanup := tmpDir(t)
	defer cleanup()
	fileName := filepath.Join(dir, "ok.json")
	writeFile(assert, fileName, []byte(`{ "test": true }`))

	for i, test := range []struct {
		body     interface{}
		expected string
	}{
		{body: &schema{"world"}, expected: `{"hello":"world"}`},
		{body: File(fileName), expected: `{"test":true}`},
	} {
		response, err := NewJsonResponse(200, test.body)
		if !assert.CmpNoError(err, "#%d", i) ||
			!assert.Cmp(response.StatusCode, 200, "#%d", i) ||
			!assert.Cmp(response.Header.Get("Content-Type"), "application/json", "#%d", i) {
			continue
		}

		assertBody(assert, response, test.expected)
	}

	// Error case
	response, err := NewJsonResponse(200, func() {})
	assert.CmpError(err)
	assert.Nil(response)
}

func checkResponder(t testing.TB, r Responder, expectedStatus int, expectedBody string) {
	helper(t).Helper()

	req, _ := http.NewRequest(http.MethodGet, "/foo", nil)
	resp, err := r(req)
	if !td.CmpNoError(t, err, "Responder returned no error") {
		return
	}

	if !td.CmpNotNil(t, resp, "Responder returned a non-nil response") {
		return
	}

	td.Cmp(t, resp.StatusCode, expectedStatus, "Status code is OK")
	assertBody(t, resp, expectedBody)
}

func TestNewJsonResponder(t *testing.T) {
	assert := td.Assert(t)

	assert.Run("OK", func(assert *td.T) {
		r, err := NewJsonResponder(200, map[string]int{"foo": 42})
		if assert.CmpNoError(err) {
			checkResponder(assert, r, 200, `{"foo":42}`)
		}
	})

	assert.Run("OK file", func(assert *td.T) {
		dir, cleanup := tmpDir(t)
		defer cleanup()
		fileName := filepath.Join(dir, "ok.json")
		writeFile(assert, fileName, []byte(`{  "foo"  :  42  }`))

		r, err := NewJsonResponder(200, File(fileName))
		if assert.CmpNoError(err) {
			checkResponder(assert, r, 200, `{"foo":42}`)
		}
	})

	assert.Run("Error", func(assert *td.T) {
		r, err := NewJsonResponder(200, func() {})
		assert.CmpError(err)
		assert.Nil(r)
	})

	assert.Run("OK don't panic", func(assert *td.T) {
		assert.CmpNotPanic(
			func() {
				r := NewJsonResponderOrPanic(200, map[string]int{"foo": 42})
				checkResponder(assert, r, 200, `{"foo":42}`)
			})
	})

	assert.Run("Panic", func(assert *td.T) {
		assert.CmpPanic(
			func() { NewJsonResponderOrPanic(200, func() {}) },
			td.Ignore())
	})
}

type schemaXML struct {
	Hello string `xml:"hello"`
}

func TestNewXmlResponse(t *testing.T) {
	assert := td.Assert(t)

	body := &schemaXML{"world"}

	b, err := xml.Marshal(body)
	if err != nil {
		t.Fatalf("Cannot xml.Marshal expected body: %s", err)
	}
	expectedBody := string(b)

	dir, cleanup := tmpDir(t)
	defer cleanup()
	fileName := filepath.Join(dir, "ok.xml")
	writeFile(assert, fileName, b)

	for i, test := range []struct {
		body     interface{}
		expected string
	}{
		{body: body, expected: expectedBody},
		{body: File(fileName), expected: expectedBody},
	} {
		response, err := NewXmlResponse(200, test.body)
		if !assert.CmpNoError(err, "#%d", i) ||
			!assert.Cmp(response.StatusCode, 200, "#%d", i) ||
			!assert.Cmp(response.Header.Get("Content-Type"), "application/xml", "#%d", i) {
			continue
		}

		assertBody(assert, response, test.expected)
	}

	// Error case
	response, err := NewXmlResponse(200, func() {})
	assert.CmpError(err)
	assert.Nil(response)
}

func TestNewXmlResponder(t *testing.T) {
	assert, require := td.AssertRequire(t)

	body := &schemaXML{"world"}

	b, err := xml.Marshal(body)
	require.CmpNoError(err)
	expectedBody := string(b)

	assert.Run("OK", func(assert *td.T) {
		r, err := NewXmlResponder(200, body)
		if assert.CmpNoError(err) {
			checkResponder(assert, r, 200, expectedBody)
		}
	})

	assert.Run("OK file", func(assert *td.T) {
		dir, cleanup := tmpDir(t)
		defer cleanup()
		fileName := filepath.Join(dir, "ok.xml")
		writeFile(assert, fileName, b)

		r, err := NewXmlResponder(200, File(fileName))
		if assert.CmpNoError(err) {
			checkResponder(assert, r, 200, expectedBody)
		}
	})

	assert.Run("Error", func(assert *td.T) {
		r, err := NewXmlResponder(200, func() {})
		assert.CmpError(err)
		assert.Nil(r)
	})

	assert.Run("OK don't panic", func(assert *td.T) {
		assert.CmpNotPanic(
			func() {
				r := NewXmlResponderOrPanic(200, body)
				checkResponder(assert, r, 200, expectedBody)
			})
	})

	assert.Run("Panic", func(assert *td.T) {
		assert.CmpPanic(
			func() { NewXmlResponderOrPanic(200, func() {}) },
			td.Ignore())
	})
}

func TestNewErrorResponder(t *testing.T) {
	assert, require := td.AssertRequire(t)

	// From go1.13, a stack frame is stored into errors issued by errors.New()
	origError := errors.New("oh no")
	responder := NewErrorResponder(origError)

	req, err := http.NewRequest(http.MethodGet, testURL, nil)
	require.CmpNoError(err)

	response, err := responder(req)
	assert.Cmp(err, origError)
	assert.Nil(response)
}

func TestResponseBody(t *testing.T) {
	assert := td.Assert(t)

	const (
		body   = "hello world"
		status = 200
	)

	assert.Run("http.Response", func(assert *td.T) {
		for i, response := range []*http.Response{
			NewBytesResponse(status, []byte(body)),
			NewStringResponse(status, body),
		} {
			assert.Run(fmt.Sprintf("resp #%d", i), func(assert *td.T) {
				assertBody(assert, response, body)

				assert.Cmp(response.StatusCode, status)

				var buf [1]byte
				_, err := response.Body.Read(buf[:])
				assert.Cmp(err, io.EOF)
			})
		}
	})

	assert.Run("Responder", func(assert *td.T) {
		for i, responder := range []Responder{
			NewBytesResponder(200, []byte(body)),
			NewStringResponder(200, body),
		} {
			assert.Run(fmt.Sprintf("resp #%d", i), func(assert *td.T) {
				req, _ := http.NewRequest("GET", "http://foo.bar", nil)
				response, err := responder(req)
				if !assert.CmpNoError(err) {
					return
				}

				assertBody(assert, response, body)

				var buf [1]byte
				_, err = response.Body.Read(buf[:])
				assert.Cmp(err, io.EOF)
			})
		}
	})
}

func TestResponder(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://foo.bar", nil)
	td.Require(t).CmpNoError(err)

	resp := &http.Response{}

	chk := func(r Responder, expectedResp *http.Response, expectedErr string) {
		helper(t).Helper()
		gotResp, gotErr := r(req)
		td.Cmp(t, gotResp, expectedResp)
		var gotErrStr string
		if gotErr != nil {
			gotErrStr = gotErr.Error()
		}
		td.Cmp(t, gotErrStr, expectedErr)
	}
	called := false
	chkNotCalled := func() {
		if called {
			helper(t).Helper()
			t.Errorf("Original responder should not be called")
			called = false
		}
	}
	chkCalled := func() {
		if !called {
			helper(t).Helper()
			t.Errorf("Original responder should be called")
		}
		called = false
	}

	r := Responder(func(*http.Request) (*http.Response, error) {
		called = true
		return resp, nil
	})
	chk(r, resp, "")
	chkCalled()

	//
	// Once
	ro := r.Once()
	chk(ro, resp, "")
	chkCalled()

	chk(ro, nil, "Responder not found for GET http://foo.bar (coz Once and already called 2 times)")
	chkNotCalled()

	chk(ro, nil, "Responder not found for GET http://foo.bar (coz Once and already called 3 times)")
	chkNotCalled()

	ro = r.Once(func(args ...interface{}) {})
	chk(ro, resp, "")
	chkCalled()

	chk(ro, nil, "Responder not found for GET http://foo.bar (coz Once and already called 2 times)")
	chkNotCalled()

	//
	// Times
	rt := r.Times(2)
	chk(rt, resp, "")
	chkCalled()

	chk(rt, resp, "")
	chkCalled()

	chk(rt, nil, "Responder not found for GET http://foo.bar (coz Times and already called 3 times)")
	chkNotCalled()

	chk(rt, nil, "Responder not found for GET http://foo.bar (coz Times and already called 4 times)")
	chkNotCalled()

	rt = r.Times(1, func(args ...interface{}) {})
	chk(rt, resp, "")
	chkCalled()

	chk(rt, nil, "Responder not found for GET http://foo.bar (coz Times and already called 2 times)")
	chkNotCalled()

	//
	// Trace
	rt = r.Trace(func(args ...interface{}) {})
	chk(rt, resp, "")
	chkCalled()

	chk(rt, resp, "")
	chkCalled()

	//
	// Delay
	rt = r.Delay(100 * time.Millisecond)
	before := time.Now()
	chk(rt, resp, "")
	duration := time.Since(before)
	chkCalled()
	td.Cmp(t, duration, td.Gte(100*time.Millisecond), "Responder is delayed")
}

func TestResponder_Then(t *testing.T) {
	assert, require := td.AssertRequire(t)

	req, err := http.NewRequest(http.MethodGet, "http://foo.bar", nil)
	require.CmpNoError(err)

	//
	// Then
	var stack string
	newResponder := func(level string) Responder {
		return func(*http.Request) (*http.Response, error) {
			stack += level
			return NewStringResponse(200, level), nil
		}
	}
	var rt Responder
	chk := func(assert *td.T, expectedLevel, expectedStack string) {
		helper(assert).Helper()
		resp, err := rt(req)
		if !assert.CmpNoError(err, "Responder call") {
			return
		}
		b, err := ioutil.ReadAll(resp.Body)
		if !assert.CmpNoError(err, "Read response") {
			return
		}
		assert.String(b, expectedLevel)
		assert.Cmp(stack, expectedStack)
	}

	A, B, C := newResponder("A"), newResponder("B"), newResponder("C")
	D, E, F := newResponder("D"), newResponder("E"), newResponder("F")

	assert.Run("simple", func(assert *td.T) {
		// (r=A,then=B)
		rt = A.Then(B)

		chk(assert, "A", "A")
		chk(assert, "B", "AB")
		chk(assert, "B", "ABB")
		chk(assert, "B", "ABBB")
	})

	stack = ""

	assert.Run("simple chained", func(assert *td.T) {
		//             (r=A,then=B)
		//          (r=↑,then=C)
		//       (r=↑,then=D)
		//    (r=↑,then=E)
		// (r=↑,then=F)
		rt = A.Then(B).
			Then(C).
			Then(D).
			Then(E).
			Then(F)

		chk(assert, "A", "A")
		chk(assert, "B", "AB")
		chk(assert, "C", "ABC")
		chk(assert, "D", "ABCD")
		chk(assert, "E", "ABCDE")
		chk(assert, "F", "ABCDEF")
		chk(assert, "F", "ABCDEFF")
		chk(assert, "F", "ABCDEFFF")
	})

	stack = ""

	assert.Run("Then Responder as Then param", func(assert *td.T) {
		assert.CmpPanic(
			func() { A.Then(B.Then(C)) },
			"Then() does not accept another Then() Responder as parameter")
	})
}

func TestParallelResponder(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://foo.bar", nil)
	td.Require(t).CmpNoError(err)

	body := strings.Repeat("ABC-", 1000)

	for ir, r := range []Responder{
		NewStringResponder(200, body),
		NewBytesResponder(200, []byte(body)),
	} {
		var wg sync.WaitGroup
		for n := 0; n < 100; n++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				resp, _ := r(req)
				b, err := ioutil.ReadAll(resp.Body)
				td.CmpNoError(t, err, "resp #%d", ir)
				td.CmpLen(t, b, 4000, "resp #%d", ir)
				td.CmpHasPrefix(t, b, "ABC-", "resp #%d", ir)
			}()
		}
		wg.Wait()
	}
}
