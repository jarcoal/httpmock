/*
Package httpmock provides tools for mocking HTTP responses.

Simple Example:
  func TestFetchArticles(t *testing.T) {
  	httpmock.Activate()
  	defer httpmock.DeactivateAndReset()

  	// Exact URL match
  	httpmock.RegisterResponder(http.MethodGet, "https://api.mybiz.com/articles",
  		httpmock.NewStringResponder(http.StatusOK, `[{"id": 1, "name": "My Great Article"}]`))

  	// Regexp match (could use httpmock.RegisterRegexpResponder instead)
  	httpmock.RegisterResponder(http.MethodGet, `=~^https://api\.mybiz\.com/articles/id/\d+\z`,
  		httpmock.NewStringResponder(http.StatusOK, `{"id": 1, "name": "My Great Article"}`))

  	// do stuff that makes a request to articles

  	// get count info
  	httpmock.GetTotalCallCount()

  	// get the amount of calls for the registered responder
  	info := httpmock.GetCallCountInfo()
  	info["GET https://api.mybiz.com/articles"]             // number of GET calls made to https://api.mybiz.com/articles
  	info["GET https://api.mybiz.com/articles/id/12"]       // number of GET calls made to https://api.mybiz.com/articles/id/12
  	info[`GET =~^https://api\.mybiz\.com/articles/id/\d+\z`] // number of GET calls made to https://api.mybiz.com/articles/id/<any-number>
  }

Advanced Example:
  func TestFetchArticles(t *testing.T) {
  	httpmock.Activate()
  	defer httpmock.DeactivateAndReset()

  	// our database of articles
  	articles := make([]map[string]interface{}, 0)

  	// mock to list out the articles
  	httpmock.RegisterResponder(http.MethodGet, "https://api.mybiz.com/articles",
  		func(req *http.Request) (*http.Response, error) {
  			resp, err := httpmock.NewJsonResponse(http.StatusOK, articles)
  			if err != nil {
  				return httpmock.NewStringResponse(http.StatusInternalServerError, ""), nil
  			}
  			return resp, nil
  		},
  	)

  	// return an article related to the request with the help of regexp submatch (\d+)
  	httpmock.RegisterResponder(http.MethodGet, `=~^https://api\.mybiz\.com/articles/id/(\d+)\z`,
  		func(req *http.Request) (*http.Response, error) {
  			// Get ID from request
  			id := httpmock.MustGetSubmatchAsUint(req, 1) // 1=first regexp submatch
  			return httpmock.NewJsonResponse(http.StatusOK, map[string]interface{}{
  				"id":   id,
  				"name": "My Great Article",
  			})
  		},
  	)

  	// mock to add a new article
  	httpmock.RegisterResponder(http.MethodPost, "https://api.mybiz.com/articles",
  		func(req *http.Request) (*http.Response, error) {
  			article := make(map[string]interface{})
  			if err := json.NewDecoder(req.Body).Decode(&article); err != nil {
  				return httpmock.NewStringResponse(http.StatusBadRequest, ""), nil
  			}

  			articles = append(articles, article)

  			resp, err := httpmock.NewJsonResponse(http.StatusOK, article)
  			if err != nil {
  				return httpmock.NewStringResponse(http.StatusInternalServerError, ""), nil
  			}
  			return resp, nil
  		},
  	)

  	// do stuff that adds and checks articles
  }
*/
package httpmock
