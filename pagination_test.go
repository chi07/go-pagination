package pagination_test

import (
	"crypto/tls"
	"net/http"
	"strings"
	"testing"

	"github.com/chi07/pagination" // Đảm bảo đúng module path
)

// =========================================================================================
// Paginator (Core Logic) Tests
// =========================================================================================

type paginatorTestCase struct {
	name                string
	totalItems          int64
	currentPage         int64
	limit               int64
	expectedPerPage     int64
	expectedCurrentPage int64
	expectedTotalItems  int64
	expectedTotalPages  int64
	expectedOffset      int64
	expectedItemCount   int64
	expectedHasPrevious bool
	expectedHasNext     bool
	expectedPrevPage    int64
	expectedNextPage    int64
}

func checkPaginator(t *testing.T, p *pagination.Paginator, tc paginatorTestCase) {
	if p.PerPage != tc.expectedPerPage {
		t.Errorf("PerPage got %d, want %d", p.PerPage, tc.expectedPerPage)
	}
	if p.CurrentPage != tc.expectedCurrentPage {
		t.Errorf("CurrentPage got %d, want %d", p.CurrentPage, tc.expectedCurrentPage)
	}
	if p.TotalItems != tc.expectedTotalItems {
		t.Errorf("TotalItems got %d, want %d", p.TotalItems, tc.expectedTotalItems)
	}
	if p.TotalPages != tc.expectedTotalPages {
		t.Errorf("TotalPages got %d, want %d", p.TotalPages, tc.expectedTotalPages)
	}
	if p.Offset != tc.expectedOffset {
		t.Errorf("Offset got %d, want %d", p.Offset, tc.expectedOffset)
	}
	if p.ItemCount != tc.expectedItemCount {
		t.Errorf("ItemCount got %d, want %d", p.ItemCount, tc.expectedItemCount)
	}
	if p.HasPrevious != tc.expectedHasPrevious {
		t.Errorf("HasPrevious got %t, want %t", p.HasPrevious, tc.expectedHasPrevious)
	}
	if p.HasNext != tc.expectedHasNext {
		t.Errorf("HasNext got %t, want %t", p.HasNext, tc.expectedHasNext)
	}
	if p.PrevPage != tc.expectedPrevPage {
		t.Errorf("PrevPage got %d, want %d", p.PrevPage, tc.expectedPrevPage)
	}
	if p.NextPage != tc.expectedNextPage {
		t.Errorf("NextPage got %d, want %d", p.NextPage, tc.expectedNextPage)
	}
}

func TestNewPaginator(t *testing.T) {
	testCases := []paginatorTestCase{
		{
			name:       "Case 1: Trang giữa",
			totalItems: 100, currentPage: 5, limit: 10,
			expectedPerPage: 10, expectedCurrentPage: 5, expectedTotalItems: 100,
			expectedTotalPages: 10, expectedOffset: 40, expectedItemCount: 10,
			expectedHasPrevious: true, expectedHasNext: true, expectedPrevPage: 4, expectedNextPage: 6,
		},
		{
			name:       "Case 5: Limit = 0 (dùng default 10)",
			totalItems: 25, currentPage: 1, limit: 0,
			expectedPerPage: 10, expectedCurrentPage: 1, expectedTotalItems: 25,
			expectedTotalPages: 3, expectedOffset: 0, expectedItemCount: 10,
			expectedHasPrevious: false, expectedHasNext: true, expectedPrevPage: 0, expectedNextPage: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := pagination.NewPaginator(tc.totalItems, tc.currentPage, tc.limit)
			checkPaginator(t, p, tc)
		})
	}
}

// =========================================================================================
// URL Builder and View Model Tests
// =========================================================================================

func mockHTTPRequest(uri, host string, headers map[string]string) *http.Request {
	req, _ := http.NewRequest("GET", uri, nil)

	if host != "" {
		req.Host = host
	} else {
		req.Host = "localhost:8080"
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	isHTTPS := req.URL.Scheme == "https" || strings.HasSuffix(req.Host, ":443")

	if isHTTPS {
		req.TLS = &tls.ConnectionState{}
	}

	return req
}

func TestBuildPageURL(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		host     string
		page     int
		opts     *pagination.BuildOptions
		headers  map[string]string
		expected string
	}{
		{
			name:     "Relative URL, giữ query mặc định",
			uri:      "http://localhost/api/items?q=test&limit=20&page=5",
			page:     3,
			opts:     nil,
			expected: "/api/items?limit=20&page=3&q=test", // Alphabetical order: limit, page, q
		},
		{
			name: "Absolute URL, dùng X-Forwarded headers",
			uri:  "http://local/api/items?q=test",
			host: "local",
			page: 2,
			opts: &pagination.BuildOptions{Mode: pagination.Absolute},
			headers: map[string]string{
				"X-Forwarded-Proto": "https",
				"X-Forwarded-Host":  "api.example.com",
			},
			expected: "https://api.example.com/api/items?page=2&q=test", // Alphabetical order: page, q
		},
		{
			name:     "Absolute URL, dùng r.Host và Scheme mặc định (HTTP)",
			uri:      "http://local/products",
			host:     "api.mysite.com", // r.Host
			page:     4,
			opts:     &pagination.BuildOptions{Mode: pagination.Absolute},
			expected: "http://api.mysite.com/products?page=4",
		},
		{
			name:     "Absolute URL, HTTPS từ URI (dùng r.TLS)",
			uri:      "https://secure.com/data",
			host:     "secure.com",
			page:     1,
			opts:     &pagination.BuildOptions{Mode: pagination.Absolute},
			expected: "https://secure.com/data?page=1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := mockHTTPRequest(tt.uri, tt.host, tt.headers)

			got := pagination.BuildPageURL(r, tt.page, tt.opts)
			if got != tt.expected {
				t.Errorf("BuildPageURL() got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNewView(t *testing.T) {
	r := mockHTTPRequest("http://localhost:8080/list?limit=10&foo=bar", "localhost:8080", nil)

	tests := []struct {
		name          string
		current       int
		total         int
		window        int
		expectedPages []int
		expectedPrev  string
		expectedNext  string
	}{
		{
			name:    "Window = 0 (full render)",
			current: 5, total: 10, window: 0,
			expectedPages: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			expectedPrev:  "/list?foo=bar&limit=10&page=4",
			expectedNext:  "/list?foo=bar&limit=10&page=6",
		},
		{
			name:    "Window = 5 (trang cuối)",
			current: 9, total: 10, window: 5,
			expectedPages: []int{6, 7, 8, 9, 10},
			expectedPrev:  "/list?foo=bar&limit=10&page=8",
			expectedNext:  "/list?foo=bar&limit=10&page=10",
		},
		{
			name:    "Current = Total (trang cuối cùng)",
			current: 10, total: 10, window: 5,
			expectedPages: []int{6, 7, 8, 9, 10},
			expectedPrev:  "/list?foo=bar&limit=10&page=9",
			expectedNext:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := pagination.NewView(r, tt.current, tt.total, nil, tt.window)

			if view.PrevURL != tt.expectedPrev {
				t.Errorf("PrevURL got %q, want %q", view.PrevURL, tt.expectedPrev)
			}
			if view.NextURL != tt.expectedNext {
				t.Errorf("NextURL got %q, want %q", view.NextURL, tt.expectedNext)
			}

			if len(view.Pages) != len(tt.expectedPages) {
				t.Fatalf("Pages length got %d, want %d", len(view.Pages), len(tt.expectedPages))
			}
		})
	}
}
