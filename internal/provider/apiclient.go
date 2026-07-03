package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	dtrack "github.com/DependencyTrack/client-go"
)

// apiErrorBodyLimit caps how much of a non-2xx response body is retained on
// an *apiError, so a misbehaving endpoint can't blow up diagnostic messages.
const apiErrorBodyLimit = 1024

// apiClient is a small shared HTTP helper for Dependency-Track endpoints not
// (yet, or ever) covered by client-go's typed methods. It centralizes base
// URL handling, authentication headers, JSON marshaling/unmarshaling, and
// pagination so individual resources don't each reimplement doRequest.
type apiClient struct {
	baseURL     string // endpoint without a trailing slash
	apiKey      string
	bearerToken string
	httpClient  *http.Client
}

// newAPIClient builds an apiClient for the given endpoint and credentials.
// Exactly one of apiKey/bearerToken is expected to be non-empty, mirroring
// the provider's mutually-exclusive authentication modes; apiKey takes
// precedence if both happen to be set.
func newAPIClient(endpoint, apiKey, bearerToken string) *apiClient {
	return &apiClient{
		baseURL:     strings.TrimSuffix(endpoint, "/"),
		apiKey:      apiKey,
		bearerToken: bearerToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// apiError is returned by apiClient.Do (and, transitively, apiGetAllPages)
// when the server responds with a non-2xx status code.
type apiError struct {
	StatusCode int
	Body       string // response body, truncated to ~1KB
}

func (e *apiError) Error() string {
	return fmt.Sprintf("dependency-track api error: status %d: %s", e.StatusCode, e.Body)
}

// isNotFound reports whether err represents an HTTP 404 response, whether it
// originated from apiClient (as *apiError) or from client-go's typed methods
// (as a dtrack.APIError, which client-go returns as *dtrack.APIError but
// which also satisfies error as a bare value since Error() has a value
// receiver).
func isNotFound(err error) bool {
	return apiErrorStatusCode(err) == http.StatusNotFound
}

// isForbidden reports whether err represents an HTTP 403 response. See
// isNotFound for the shapes of error it recognizes.
func isForbidden(err error) bool {
	return apiErrorStatusCode(err) == http.StatusForbidden
}

// apiErrorStatusCode extracts a status code from err if it is (or wraps) an
// *apiError or a dtrack.APIError (pointer or value form), and -1 otherwise.
func apiErrorStatusCode(err error) int {
	var ae *apiError
	if errors.As(err, &ae) {
		return ae.StatusCode
	}

	var dtErrPtr *dtrack.APIError
	if errors.As(err, &dtErrPtr) {
		return dtErrPtr.StatusCode
	}

	var dtErr dtrack.APIError
	if errors.As(err, &dtErr) {
		return dtErr.StatusCode
	}

	return -1
}

// Do performs an HTTP request against path (joined to the client's baseURL)
// and decodes a JSON response into out.
//
//   - body is marshaled as JSON when non-nil; Content-Type and Accept are
//     always set to application/json.
//   - Authentication uses X-Api-Key when an API key is configured, otherwise
//     Authorization: Bearer <token>.
//   - A non-2xx response yields an *apiError carrying the status code and a
//     truncated copy of the response body.
//   - out is decoded from the response body when non-nil, the status isn't
//     204 No Content, and the body is non-empty.
func (c *apiClient) Do(ctx context.Context, method, path string, body, out any) error {
	_, err := c.doWithHeaders(ctx, method, path, body, out)
	return err
}

// doWithHeaders is Do plus the response headers, so callers like
// apiGetAllPages can inspect pagination headers (e.g. X-Total-Count) without
// duplicating the request/response plumbing.
func (c *apiClient) doWithHeaders(ctx context.Context, method, path string, body, out any) (http.Header, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	switch {
	case c.apiKey != "":
		req.Header.Set("X-Api-Key", c.apiKey)
	case c.bearerToken != "":
		req.Header.Set("Authorization", "Bearer "+c.bearerToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.Header, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.Header, &apiError{StatusCode: resp.StatusCode, Body: truncateBody(respBody)}
	}

	if out != nil && resp.StatusCode != http.StatusNoContent && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return resp.Header, fmt.Errorf("decode response body: %w", err)
		}
	}

	return resp.Header, nil
}

// truncateBody returns b as a string, capped to apiErrorBodyLimit bytes.
func truncateBody(b []byte) string {
	if len(b) <= apiErrorBodyLimit {
		return string(b)
	}
	return string(b[:apiErrorBodyLimit]) + "...(truncated)"
}

// apiGetAllPagesSafetyCap bounds how many pages apiGetAllPages will fetch
// before giving up, so a server that never signals completion (e.g. a buggy
// X-Total-Count or an endpoint that always returns a full page) can't spin
// forever.
const apiGetAllPagesSafetyCap = 1000

// apiGetAllPages fetches every page of a Dependency-Track list endpoint at
// path, merging query into each page's request alongside pageNumber/pageSize.
//
// DT v5 enforces a pageSize cap of 100; this always requests pages of 100 so
// the same code works against v4 and v5. Pagination stops when the number of
// items collected reaches X-Total-Count (if the header is present and
// parses), or otherwise as soon as a page comes back short of pageSize
// (including empty), which also correctly terminates a fetch of exactly N*100
// items.
func apiGetAllPages[T any](ctx context.Context, c *apiClient, path string, query url.Values) ([]T, error) {
	const pageSize = 100

	base := cloneQueryValues(query)
	base.Set("pageSize", strconv.Itoa(pageSize))

	var all []T
	total := -1

	for page := 1; page <= apiGetAllPagesSafetyCap; page++ {
		q := cloneQueryValues(base)
		q.Set("pageNumber", strconv.Itoa(page))

		var items []T
		headers, err := c.doWithHeaders(ctx, http.MethodGet, path+"?"+q.Encode(), nil, &items)
		if err != nil {
			return nil, err
		}

		all = append(all, items...)

		if total < 0 {
			if tc := headers.Get("X-Total-Count"); tc != "" {
				if n, convErr := strconv.Atoi(tc); convErr == nil {
					total = n
				}
			}
		}

		if total >= 0 && len(all) >= total {
			return all, nil
		}

		if len(items) < pageSize {
			return all, nil
		}
	}

	return nil, fmt.Errorf("apiGetAllPages: exceeded safety cap of %d pages fetching %s", apiGetAllPagesSafetyCap, path)
}

// cloneQueryValues returns a copy of v so callers can mutate the result
// without affecting the caller's url.Values (or a shared base map across
// pagination loop iterations).
func cloneQueryValues(v url.Values) url.Values {
	clone := make(url.Values, len(v))
	for k, vals := range v {
		clone[k] = append([]string(nil), vals...)
	}
	return clone
}
