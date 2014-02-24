package httpmock

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

func TestMockTransport(t *testing.T) {
	Activate()
	defer Deactivate()

	url := "https://github.com/"
	body := "hello world"

	RegisterResponder("GET", url, NewStringResponder(body, 200))

	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != body {
		t.FailNow()
	}

	// the http client wraps our NoResponderFound error, so we just try and match on text
	if _, err := http.Get("http://not-registered.com/"); !strings.Contains(err.Error(),
		NoResponderFound.Error()) {

		t.Fatal(err)
	}
}
