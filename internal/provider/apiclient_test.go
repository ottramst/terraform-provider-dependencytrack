package provider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	dtrack "github.com/DependencyTrack/client-go"
)

type apiClientTestItem struct {
	ID int `json:"id"`
}

func TestNewAPIClient(t *testing.T) {
	c := newAPIClient("https://dtrack.example.com/", "key", "")

	if c.baseURL != "https://dtrack.example.com" {
		t.Errorf("newAPIClient baseURL = %q, want trailing slash trimmed", c.baseURL)
	}
	if c.apiKey != "key" {
		t.Errorf("newAPIClient apiKey = %q, want %q", c.apiKey, "key")
	}
	if c.httpClient == nil {
		t.Fatal("newAPIClient httpClient is nil")
	}
	if c.httpClient.Timeout <= 0 {
		t.Errorf("newAPIClient httpClient.Timeout = %v, want > 0", c.httpClient.Timeout)
	}
}

func TestAPIClientDo_Success(t *testing.T) {
	var gotMethod, gotPath, gotAuth, gotContentType, gotAccept string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("X-Api-Key")
		gotContentType = r.Header.Get("Content-Type")
		gotAccept = r.Header.Get("Accept")

		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": 42}`))
	}))
	defer srv.Close()

	c := newAPIClient(srv.URL, "test-key", "")

	var out apiClientTestItem
	err := c.Do(context.Background(), http.MethodPost, "/api/v1/thing", map[string]any{"name": "widget"}, &out)
	if err != nil {
		t.Fatalf("Do returned unexpected error: %s", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/api/v1/thing" {
		t.Errorf("path = %q, want /api/v1/thing", gotPath)
	}
	if gotAuth != "test-key" {
		t.Errorf("X-Api-Key header = %q, want test-key", gotAuth)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type header = %q, want application/json", gotContentType)
	}
	if gotAccept != "application/json, application/problem+json;q=0.9" {
		t.Errorf("Accept header = %q, want application/json, application/problem+json;q=0.9", gotAccept)
	}
	if gotBody["name"] != "widget" {
		t.Errorf("request body name = %v, want widget", gotBody["name"])
	}
	if out.ID != 42 {
		t.Errorf("decoded out.ID = %d, want 42", out.ID)
	}
}

func TestAPIClientDo_AuthHeaderSelection(t *testing.T) {
	tests := []struct {
		name        string
		apiKey      string
		bearerToken string
		wantAPIKey  string
		wantBearer  string
	}{
		{name: "api key wins when set", apiKey: "the-key", bearerToken: "the-token", wantAPIKey: "the-key", wantBearer: ""},
		{name: "bearer token used when no api key", apiKey: "", bearerToken: "the-token", wantAPIKey: "", wantBearer: "Bearer the-token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotAPIKey, gotAuthorization string

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotAPIKey = r.Header.Get("X-Api-Key")
				gotAuthorization = r.Header.Get("Authorization")
				w.WriteHeader(http.StatusNoContent)
			}))
			defer srv.Close()

			c := newAPIClient(srv.URL, tt.apiKey, tt.bearerToken)
			if err := c.Do(context.Background(), http.MethodGet, "/api/v1/thing", nil, nil); err != nil {
				t.Fatalf("Do returned unexpected error: %s", err)
			}

			if gotAPIKey != tt.wantAPIKey {
				t.Errorf("X-Api-Key header = %q, want %q", gotAPIKey, tt.wantAPIKey)
			}
			if gotAuthorization != tt.wantBearer {
				t.Errorf("Authorization header = %q, want %q", gotAuthorization, tt.wantBearer)
			}
		})
	}
}

func TestAPIClientDo_NoContentSkipsDecode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newAPIClient(srv.URL, "key", "")

	var out apiClientTestItem
	if err := c.Do(context.Background(), http.MethodDelete, "/api/v1/thing", nil, &out); err != nil {
		t.Fatalf("Do returned unexpected error: %s", err)
	}
}

func TestAPIClientDo_ErrorStatusCodes(t *testing.T) {
	tests := []struct {
		name          string
		status        int
		wantNotFound  bool
		wantForbidden bool
	}{
		{name: "404 is not found", status: http.StatusNotFound, wantNotFound: true, wantForbidden: false},
		{name: "403 is forbidden", status: http.StatusForbidden, wantNotFound: false, wantForbidden: true},
		{name: "500 is neither", status: http.StatusInternalServerError, wantNotFound: false, wantForbidden: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte("boom"))
			}))
			defer srv.Close()

			c := newAPIClient(srv.URL, "key", "")

			err := c.Do(context.Background(), http.MethodGet, "/api/v1/thing", nil, nil)
			if err == nil {
				t.Fatal("Do returned nil error for a non-2xx status")
			}

			if got := isNotFound(err); got != tt.wantNotFound {
				t.Errorf("isNotFound(err) = %v, want %v (err: %s)", got, tt.wantNotFound, err)
			}
			if got := isForbidden(err); got != tt.wantForbidden {
				t.Errorf("isForbidden(err) = %v, want %v (err: %s)", got, tt.wantForbidden, err)
			}
		})
	}
}

func TestAPIClientDo_ErrorBodyTruncated(t *testing.T) {
	hugeBody := make([]byte, 5000)
	for i := range hugeBody {
		hugeBody[i] = 'x'
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(hugeBody)
	}))
	defer srv.Close()

	c := newAPIClient(srv.URL, "key", "")

	err := c.Do(context.Background(), http.MethodGet, "/api/v1/thing", nil, nil)
	if err == nil {
		t.Fatal("expected an error")
	}

	ae, ok := err.(*apiError)
	if !ok {
		t.Fatalf("expected *apiError, got %T: %s", err, err)
	}

	if len(ae.Body) > 1200 {
		t.Errorf("apiError.Body length = %d, want truncated to ~1KB", len(ae.Body))
	}
}

func TestIsNotFound_IsForbidden_AgainstDtrackAPIError(t *testing.T) {
	// client-go's checkResponseForError returns a *dtrack.APIError, but APIError's
	// Error() method has a value receiver, so a bare dtrack.APIError value also
	// satisfies the error interface. Cover both shapes.
	var notFoundPtr error = &dtrack.APIError{StatusCode: http.StatusNotFound, Message: "not found"}
	if !isNotFound(notFoundPtr) {
		t.Error("isNotFound(*dtrack.APIError 404) = false, want true")
	}
	if isForbidden(notFoundPtr) {
		t.Error("isForbidden(*dtrack.APIError 404) = true, want false")
	}

	var forbiddenValue error = dtrack.APIError{StatusCode: http.StatusForbidden, Message: "forbidden"}
	if !isForbidden(forbiddenValue) {
		t.Error("isForbidden(dtrack.APIError 403 value) = false, want true")
	}
	if isNotFound(forbiddenValue) {
		t.Error("isNotFound(dtrack.APIError 403 value) = true, want false")
	}
}

func TestApiGetAllPages_MultiPageWithTotalCount(t *testing.T) {
	const total = 250

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("pageSize"); got != "100" {
			t.Errorf("pageSize = %q, want 100", got)
		}
		page, err := strconv.Atoi(q.Get("pageNumber"))
		if err != nil {
			t.Fatalf("invalid pageNumber: %s", q.Get("pageNumber"))
		}
		if got := q.Get("filter"); got != "active" {
			t.Errorf("filter query param = %q, want active (caller params should be preserved)", got)
		}

		start := (page - 1) * 100
		end := start + 100
		if end > total {
			end = total
		}

		var items []apiClientTestItem
		for i := start; i < end; i++ {
			items = append(items, apiClientTestItem{ID: i})
		}

		w.Header().Set("X-Total-Count", strconv.Itoa(total))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(items)
	}))
	defer srv.Close()

	c := newAPIClient(srv.URL, "key", "")

	got, err := apiGetAllPages[apiClientTestItem](context.Background(), c, "/api/v1/thing", url.Values{"filter": []string{"active"}})
	if err != nil {
		t.Fatalf("apiGetAllPages returned unexpected error: %s", err)
	}

	if len(got) != total {
		t.Fatalf("apiGetAllPages returned %d items, want %d", len(got), total)
	}
	for i, item := range got {
		if item.ID != i {
			t.Fatalf("item[%d].ID = %d, want %d", i, item.ID, i)
		}
	}
}

func TestApiGetAllPages_ShortPageTerminatesWithoutHeader(t *testing.T) {
	const total = 130

	var requestCount int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		q := r.URL.Query()
		page, _ := strconv.Atoi(q.Get("pageNumber"))

		start := (page - 1) * 100
		end := start + 100
		if end > total {
			end = total
		}
		if start > total {
			start = total
		}

		var items []apiClientTestItem
		for i := start; i < end; i++ {
			items = append(items, apiClientTestItem{ID: i})
		}

		// Deliberately no X-Total-Count header.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(items)
	}))
	defer srv.Close()

	c := newAPIClient(srv.URL, "key", "")

	got, err := apiGetAllPages[apiClientTestItem](context.Background(), c, "/api/v1/thing", nil)
	if err != nil {
		t.Fatalf("apiGetAllPages returned unexpected error: %s", err)
	}

	if len(got) != total {
		t.Fatalf("apiGetAllPages returned %d items, want %d", len(got), total)
	}
	if requestCount != 2 {
		t.Fatalf("apiGetAllPages made %d requests, want 2 (100 then a short 30-item page)", requestCount)
	}
}

func TestFetchAllPages_PaginatesWithoutTotalCount(t *testing.T) {
	// Mirrors client-go's UserService.GetAllManaged, which never populates
	// Page.TotalCount. fetchAllPages must still fetch every page (dtrack.ForEach
	// would stop after the first because itemsSeen >= TotalCount(0)).
	const total = 250

	var gotPageSizes []int
	fetch := func(_ context.Context, po dtrack.PageOptions) (dtrack.Page[apiClientTestItem], error) {
		gotPageSizes = append(gotPageSizes, po.PageSize)

		start := (po.PageNumber - 1) * po.PageSize
		end := start + po.PageSize
		if end > total {
			end = total
		}

		var items []apiClientTestItem
		for i := start; i < end; i++ {
			items = append(items, apiClientTestItem{ID: i})
		}

		// Deliberately leave TotalCount unset, as GetAllManaged does.
		return dtrack.Page[apiClientTestItem]{Items: items}, nil
	}

	got, err := fetchAllPages(context.Background(), fetch)
	if err != nil {
		t.Fatalf("fetchAllPages returned unexpected error: %s", err)
	}

	if len(got) != total {
		t.Fatalf("fetchAllPages returned %d items, want %d", len(got), total)
	}
	for i, item := range got {
		if item.ID != i {
			t.Fatalf("item[%d].ID = %d, want %d", i, item.ID, i)
		}
	}
	for _, size := range gotPageSizes {
		if size != 100 {
			t.Errorf("requested pageSize = %d, want 100", size)
		}
	}
	if len(gotPageSizes) != 3 {
		t.Errorf("fetched %d pages, want 3 (100 + 100 + short 50)", len(gotPageSizes))
	}
}

func TestFetchAllPages_PropagatesError(t *testing.T) {
	wantErr := errors.New("boom")
	fetch := func(_ context.Context, _ dtrack.PageOptions) (dtrack.Page[apiClientTestItem], error) {
		return dtrack.Page[apiClientTestItem]{}, wantErr
	}

	_, err := fetchAllPages(context.Background(), fetch)
	if !errors.Is(err, wantErr) {
		t.Fatalf("fetchAllPages error = %v, want %v", err, wantErr)
	}
}

func TestFetchAllPages_SafetyCap(t *testing.T) {
	// A server that always returns a full page must not loop forever.
	fetch := func(_ context.Context, po dtrack.PageOptions) (dtrack.Page[apiClientTestItem], error) {
		return dtrack.Page[apiClientTestItem]{Items: make([]apiClientTestItem, po.PageSize)}, nil
	}

	_, err := fetchAllPages(context.Background(), fetch)
	if err == nil {
		t.Fatal("expected an error when the safety cap is exceeded")
	}
}

func TestApiGetAllPages_SafetyCap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		items := make([]apiClientTestItem, 100)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(items)
	}))
	defer srv.Close()

	c := newAPIClient(srv.URL, "key", "")

	_, err := apiGetAllPages[apiClientTestItem](context.Background(), c, "/api/v1/thing", nil)
	if err == nil {
		t.Fatal("expected an error when the safety cap is exceeded")
	}
}
