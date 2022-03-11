package httpmock_test

import (
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/maxatome/go-testdeep/td"
)

func assertBody(t testing.TB, resp *http.Response, expected string) bool {
	defer resp.Body.Close()

	helper(t).Helper()

	data, err := ioutil.ReadAll(resp.Body)
	td.Require(t).CmpNoError(err)

	return td.CmpString(t, data, expected)
}

func tmpDir(t testing.TB) (string, func()) {
	helper(t).Helper()
	dir, err := ioutil.TempDir("", "httpmock")
	td.Require(t).CmpNoError(err)
	return dir, func() { os.RemoveAll(dir) }
}

func writeFile(t testing.TB, file string, content []byte) {
	helper(t).Helper()
	td.Require(t).CmpNoError(ioutil.WriteFile(file, content, 0644))
}

// fakeHelper allows to compensate the absence of
// (*testing.T).Helper() in go<1.9.
type fakeHelper struct{}

func (f fakeHelper) Helper() {}

type helperAble interface {
	Helper()
}

func helper(t interface{}) helperAble {
	if th, ok := t.(helperAble); ok {
		return th
	}
	return fakeHelper{}
}
