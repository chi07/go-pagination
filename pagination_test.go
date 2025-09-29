package pagination_test

import (
	"crypto/tls"
	"net/http"
	"strings"
	"testing"

	"github.com/chi07/pagination"
)

// ---------------------- Helpers ----------------------

func mockHTTPRequest(uri, host string, headers map[string]string) *http.Request {
	req, _ := http.NewRequest(http.MethodGet, uri, nil)

	if host != "" {
		req.Host = host
	} else {
		req.Host = "localhost:8080"
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Mark TLS if https or :443 to emulate secure transport
	if strings.HasPrefix(uri, "https://") || strings.HasSuffix(req.Host, ":443") {
		req.TLS = &tls.ConnectionState{}
	}
	return req
}

func want(t *testing.T, name string, got, expected any) {
	t.Helper()
	if got != expected {
		t.Fatalf("%s: got %v, want %v", name, got, expected)
	}
}

// ---------------------- Paginator Tests ----------------------

func TestNewPaginator_EdgesAndNormal(t *testing.T) {
	tests := []struct {
		name                     string
		totalItems               int64
		currentPage              int64
		limit                    int64
		wantPerPage              int64
		wantCurrentPage          int64
		wantTotalPages           int64
		wantOffset               int64
		wantItemCount            int64
		wantHasPrev, wantHasNext bool
		wantPrev, wantNext       int64
	}{
		{
			name:       "Middle page",
			totalItems: 100, currentPage: 5, limit: 10,
			wantPerPage: 10, wantCurrentPage: 5, wantTotalPages: 10,
			wantOffset: 40, wantItemCount: 10,
			wantHasPrev: true, wantHasNext: true,
			wantPrev: 4, wantNext: 6,
		},
		{
			name:       "First page",
			totalItems: 23, currentPage: 1, limit: 5,
			wantPerPage: 5, wantCurrentPage: 1, wantTotalPages: 5,
			wantOffset: 0, wantItemCount: 5,
			wantHasPrev: false, wantHasNext: true,
			wantPrev: 0, wantNext: 2,
		},
		{
			name:       "Last page partial",
			totalItems: 23, currentPage: 5, limit: 5,
			wantPerPage: 5, wantCurrentPage: 5, wantTotalPages: 5,
			wantOffset: 20, wantItemCount: 3,
			wantHasPrev: true, wantHasNext: false,
			wantPrev: 4, wantNext: 0,
		},
		{
			name:       "Clamp currentPage > totalPages",
			totalItems: 12, currentPage: 99, limit: 5,
			wantPerPage: 5, wantCurrentPage: 3, wantTotalPages: 3,
			wantOffset: 10, wantItemCount: 2,
			wantHasPrev: true, wantHasNext: false,
			wantPrev: 2, wantNext: 0,
		},
		{
			name:       "Clamp currentPage < 1",
			totalItems: 12, currentPage: 0, limit: 5,
			wantPerPage: 5, wantCurrentPage: 1, wantTotalPages: 3,
			wantOffset: 0, wantItemCount: 5,
			wantHasPrev: false, wantHasNext: true,
			wantPrev: 0, wantNext: 2,
		},
		{
			name:       "PerPage defaulted to 10 when <= 0",
			totalItems: 25, currentPage: 1, limit: 0,
			wantPerPage: 10, wantCurrentPage: 1, wantTotalPages: 3,
			wantOffset: 0, wantItemCount: 10,
			wantHasPrev: false, wantHasNext: true,
			wantPrev: 0, wantNext: 2,
		},
		{
			name:       "No items: normalized to page 1, totals=1",
			totalItems: 0, currentPage: 5, limit: 10,
			wantPerPage: 10, wantCurrentPage: 1, wantTotalPages: 1,
			wantOffset: 0, wantItemCount: 0,
			wantHasPrev: false, wantHasNext: false,
			wantPrev: 0, wantNext: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := pagination.NewPaginator(tt.totalItems, tt.currentPage, tt.limit)
			want(t, "PerPage", p.PerPage, tt.wantPerPage)
			want(t, "CurrentPage", p.CurrentPage, tt.wantCurrentPage)
			want(t, "TotalPages", p.TotalPages, tt.wantTotalPages)
			want(t, "Offset", p.Offset, tt.wantOffset)
			want(t, "ItemCount", p.ItemCount, tt.wantItemCount)
			want(t, "HasPrevious", p.HasPrevious, tt.wantHasPrev)
			want(t, "HasNext", p.HasNext, tt.wantHasNext)
			want(t, "PrevPage", p.PrevPage, tt.wantPrev)
			want(t, "NextPage", p.NextPage, tt.wantNext)
		})
	}
}

// ---------------------- URL Builder Tests ----------------------

func TestBuildPageURL_Relative_KeepQuery_Defaults(t *testing.T) {
	r := mockHTTPRequest("http://localhost/api/items?q=test&limit=20&page=5", "localhost", nil)
	got := pagination.BuildPageURL(r, 3, nil) // defaults: Relative, keep query, page param "page"
	// url.Values.Encode sorts keys alphabetically
	want(t, "RelativeKeepQuery", got, "/api/items?limit=20&page=3&q=test")
}

func TestBuildPageURL_Absolute_ForwardedHeaders(t *testing.T) {
	h := map[string]string{
		"X-Forwarded-Proto": "https",
		"X-Forwarded-Host":  "api.example.com",
	}
	r := mockHTTPRequest("http://local/api/items?q=test", "local", h)
	opts := &pagination.BuildOptions{Mode: pagination.Absolute}
	got := pagination.BuildPageURL(r, 2, opts)
	want(t, "AbsoluteForwarded", got, "https://api.example.com/api/items?page=2&q=test")
}

func TestBuildPageURL_Absolute_TLSFromRequest(t *testing.T) {
	r := mockHTTPRequest("https://secure.com/data?x=1", "secure.com", nil)
	opts := &pagination.BuildOptions{Mode: pagination.Absolute}
	got := pagination.BuildPageURL(r, 1, opts)
	want(t, "AbsoluteTLS", got, "https://secure.com/data?page=1&x=1")
}

func TestBuildPageURL_Absolute_Overrides(t *testing.T) {
	r := mockHTTPRequest("http://local/path?a=1", "local", nil)
	opts := &pagination.BuildOptions{
		Mode:              pagination.Absolute,
		Scheme:            "https",
		Host:              "override.example",
		Path:              "/custom",
		PageParam:         "pg",
		KeepExistingQuery: true,
	}
	got := pagination.BuildPageURL(r, 7, opts)
	want(t, "AbsoluteOverride", got, "https://override.example/custom?a=1&pg=7")
}

// ---------------------- View (Window) Tests ----------------------

func TestNewView_Windows_Start_Middle_End(t *testing.T) {
	r := mockHTTPRequest("http://localhost:8080/list?limit=10&foo=bar", "localhost:8080", nil)

	tests := []struct {
		name                   string
		current, total, window int
		wantPrev, wantNext     string
		wantPages              []int
		wantActiveAt           int // index within Pages expected slice
	}{
		{
			name:    "Full window = 0 (renders all)",
			current: 5, total: 10, window: 0,
			wantPrev:     "/list?foo=bar&limit=10&page=4",
			wantNext:     "/list?foo=bar&limit=10&page=6",
			wantPages:    []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			wantActiveAt: 4, // index of '5'
		},
		{
			name:    "Window 5 at start",
			current: 1, total: 10, window: 5,
			wantPrev:     "",
			wantNext:     "/list?foo=bar&limit=10&page=2",
			wantPages:    []int{1, 2, 3, 4, 5},
			wantActiveAt: 0,
		},
		{
			name:    "Window 5 middle",
			current: 6, total: 10, window: 5,
			wantPrev:     "/list?foo=bar&limit=10&page=5",
			wantNext:     "/list?foo=bar&limit=10&page=7",
			wantPages:    []int{4, 5, 6, 7, 8},
			wantActiveAt: 2,
		},
		{
			name:    "Window 5 at end",
			current: 10, total: 10, window: 5,
			wantPrev:     "/list?foo=bar&limit=10&page=9",
			wantNext:     "",
			wantPages:    []int{6, 7, 8, 9, 10},
			wantActiveAt: 4,
		},
		{
			name:    "Clamp: current < 1, total < 1 normalize",
			current: -3, total: 0, window: 5,
			wantPrev:     "",
			wantNext:     "/list?foo=bar&limit=10&page=2", // total normalized to 1, but window < total condition fails so pages=1..1
			wantPages:    []int{1},
			wantActiveAt: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := pagination.NewView(r, tt.current, tt.total, nil, tt.window)

			if view.PrevURL != tt.wantPrev {
				t.Fatalf("PrevURL got %q, want %q", view.PrevURL, tt.wantPrev)
			}

			if len(view.Pages) != len(tt.wantPages) {
				t.Fatalf("Pages length got %d, want %d", len(view.Pages), len(tt.wantPages))
			}
			for i, p := range view.Pages {
				if p.Num != tt.wantPages[i] {
					t.Fatalf("Pages[%d].Num got %d, want %d", i, p.Num, tt.wantPages[i])
				}
			}
			if !view.Pages[tt.wantActiveAt].Active {
				t.Fatalf("Active page not marked active; want index %d number %d", tt.wantActiveAt, tt.wantPages[tt.wantActiveAt])
			}
		})
	}
}
