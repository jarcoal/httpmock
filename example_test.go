package httpmock_test

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jarcoal/httpmock"
)

// ExampleActivateNonDefault mocks requests made through a custom
// *http.Client that carries its own Transport instead of relying on
// http.DefaultClient / http.DefaultTransport.
//
// httpmock never touches http.DefaultClient, so a plain Activate would
// not intercept this client. ActivateNonDefault swaps the client's
// Transport for the mock one, whatever the client was configured with,
// and DeactivateNonDefault puts the original Transport back.
func ExampleActivateNonDefault() {
	client := &http.Client{
		Transport: &http.Transport{
			ResponseHeaderTimeout: 5 * time.Second,
		},
	}

	httpmock.ActivateNonDefault(client)
	defer httpmock.DeactivateNonDefault(client)

	httpmock.RegisterResponder("GET", "https://api.mybiz.com/articles.json",
		httpmock.NewStringResponder(200, `[{"id": 1, "name": "My Great Article"}]`))

	resp, err := client.Get("https://api.mybiz.com/articles.json")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer resp.Body.Close() //nolint: errcheck

	body, _ := io.ReadAll(resp.Body)
	fmt.Println(resp.StatusCode)
	fmt.Println(string(body))
	// Output:
	// 200
	// [{"id": 1, "name": "My Great Article"}]
}
