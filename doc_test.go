package httpmock

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

func ExampleRegisterStubRequest() {
	Activate()
	defer DeactivateAndReset()

	RegisterStubRequest(
		NewStubRequest(
			"GET",
			"http://example.com/",
			NewStringResponder(200, "ok"),
		),
	)

	resp, err := http.Get("http://example.com/")
	if err != nil {
		// handle error
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// handle error
	}

	fmt.Println(string(body))

	if err = AllStubsCalled(); err != nil {
		// handle error
	}

	// Output: ok
}
