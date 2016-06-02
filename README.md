httpmock [![Build Status](https://travis-ci.org/jarcoal/httpmock.png?branch=master)](https://travis-ci.org/jarcoal/httpmock)
=====

### Simple Example:
```go
func TestFetchArticles(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterStubRequest(httpmock.NewStubRequest(
    "GET",
    "https://api.mybiz.com/articles.json",
		httpmock.NewStringResponder(200, `[{"id": 1, "name": "My Great Article"}]`),
  ))

	// do stuff that makes a request to articles.json

  // verify all registered stubs were called
  if err := httpmock.AllStubsWereCalled(); err != nil {
    t.Errorf("Not all stubs were called: %s", err)
  }
}
```

### Advanced Example:
```go
func TestFetchArticles(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// our database of articles
	articles := make([]map[string]interface{}, 0)

	// mock to list out the articles
	httpmock.RegisterStubRequest(&httpmock.StubRequest{
    Method: "GET",
    URL: "https://api.mybiz.com/articles.json",
		Responder: func(req *http.Request) (*http.Response, error) {
			resp, err := httpmock.NewJsonResponse(200, articles)
			if err != nil {
				return httpmock.NewStringResponse(500, ""), nil
			}
			return resp, nil
		},
	})

	// mock to add a new article
	httpmock.RegisterStubRequest(&httpmock.StubRequest{
    Method: "POST",
    URL: "https://api.mybiz.com/articles.json",
		Responder: func(req *http.Request) (*http.Response, error) {
			article := make(map[string]interface{})
			if err := json.NewDecoder(req.Body).Decode(&article); err != nil {
				return httpmock.NewStringResponse(400, ""), nil
			}

			articles = append(articles, article)

			resp, err := httpmock.NewJsonResponse(200, article)
			if err != nil {
				return httpmock.NewStringResponse(500, ""), nil
			}
			return resp, nil
		},
	})

	// do stuff that adds and checks articles
}
```
