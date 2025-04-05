package httpmock_test

import (
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/maxatome/go-testdeep/td"
)

func assertBody(t testing.TB, resp *http.Response, expected string) bool {
	t.Helper()

	require := td.Require(t)
	require.NotNil(resp)

	defer resp.Body.Close() //nolint: errcheck

	data, err := io.ReadAll(resp.Body)
	require.CmpNoError(err)

	return td.CmpString(t, data, expected)
}

func writeFile(t testing.TB, file string, content []byte) {
	t.Helper()
	td.Require(t).CmpNoError(os.WriteFile(file, content, 0644))
}
