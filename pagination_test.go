package pagination_test

import (
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"

	"github.com/chi07/pagination"
)

// =========================================================================================
// Paginator (Core Logic) Tests
// =========================================================================================

type paginatorTestCase struct {
	name string

	// Input
	totalItems  int64
	currentPage int64
	limit       int64

	// Expected Output
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
			name:       "Case 2: Trang cuối, dư item",
			totalItems: 42, currentPage: 5, limit: 10,
			expectedPerPage: 10, expectedCurrentPage: 5, expectedTotalItems: 42,
			expectedTotalPages: 5, expectedOffset: 40, expectedItemCount: 2,
			expectedHasPrevious: true, expectedHasNext: false, expectedPrevPage: 4, expectedNextPage: 0,
		},
		{
			name:       "Case 3: CurrentPage vượt quá giới hạn",
			totalItems: 42, currentPage: 10, limit: 10,
			expectedPerPage: 10, expectedCurrentPage: 5, // Bị giới hạn lại về 5
			expectedTotalItems: 42, expectedTotalPages: 5, expectedOffset: 40, expectedItemCount: 2,
			expectedHasPrevious: true, expectedHasNext: false, expectedPrevPage: 4, expectedNextPage: 0,
		},
		{
			name:       "Case 4: TotalItems = 0",
			totalItems: 0, currentPage: 1, limit: 10,
			expectedPerPage: 10, expectedCurrentPage: 1, expectedTotalItems: 0,
			expectedTotalPages: 1, expectedOffset: 0, expectedItemCount: 0,
			expectedHasPrevious: false, expectedHasNext: false, expectedPrevPage: 0, expectedNextPage: 0,
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
// URL Builder and VM Tests
// =========================================================================================

// Helper để tạo Fiber Context (mock)
func mockFiberCtx(method, path, body string, headers map[string]string) *fiber.Ctx {
	// 1. Khởi tạo một fasthttp.RequestCtx rỗng (chỉ cần struct)
	fctx := &fasthttp.RequestCtx{}

	// 2. Khởi tạo fasthttp.Request để mô phỏng request thực
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod(method)
	req.SetRequestURI(path)
	req.SetBodyString(body)

	// 3. Set headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// 4. Liên kết Request với fasthttp.RequestCtx
	fctx.Request = *req

	// 5. Sử dụng fiber.App để AcquireCtx (cấp phát Fiber Context)
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	c := app.AcquireCtx(fctx)

	return c
}

func TestBuildPageURL(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		page     int
		opts     *pagination.BuildOptions
		headers  map[string]string
		expected string // Đã sửa
	}{
		{
			name:     "Relative URL, giữ query mặc định",
			path:     "/api/items?q=test&limit=20&page=5",
			page:     3,
			opts:     nil,
			expected: "/api/items?limit=20&page=3&q=test",
		},
		{
			name: "Absolute URL, dùng X-Forwarded headers",
			path: "/api/items?q=test",
			page: 2,
			opts: &pagination.BuildOptions{Mode: pagination.Absolute},
			headers: map[string]string{
				"X-Forwarded-Proto": "https",
				"X-Forwarded-Host":  "api.example.com",
			},
			expected: "https://api.example.com/api/items?page=2&q=test",
		},
		{
			name: "Absolute URL, dùng c.Protocol/c.Hostname",
			path: "/products",
			page: 4,
			opts: &pagination.BuildOptions{Mode: pagination.Absolute},
			headers: map[string]string{
				"Host": "localhost:3000",
			},
			expected: "http://localhost:3000/products?page=4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := mockFiberCtx(fiber.MethodGet, tt.path, "", tt.headers)
			defer fiber.New().ReleaseCtx(c)

			got := pagination.BuildPageURL(c, tt.page, tt.opts)
			if got != tt.expected {
				t.Errorf("BuildPageURL() got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNewVM(t *testing.T) {
	// Mock Context
	c := mockFiberCtx(fiber.MethodGet, "/list?limit=10", "", nil)
	defer fiber.New().ReleaseCtx(c)

	tests := []struct {
		name          string
		current       int
		total         int
		window        int
		expectedPages []int
		expectedPrev  string // Đã sửa
		expectedNext  string // Đã sửa
	}{
		{
			name:    "Window = 0 (full render)",
			current: 5, total: 10, window: 0,
			expectedPages: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			// Sửa: 'limit' -> 'page'
			expectedPrev: "/list?limit=10&page=4",
			expectedNext: "/list?limit=10&page=6",
		},
		{
			name:    "Window = 5 (trang giữa)",
			current: 5, total: 10, window: 5,
			expectedPages: []int{3, 4, 5, 6, 7},
			expectedPrev:  "/list?limit=10&page=4",
			expectedNext:  "/list?limit=10&page=6",
		},
		{
			name:    "Window = 5 (trang cuối)",
			current: 9, total: 10, window: 5,
			expectedPages: []int{6, 7, 8, 9, 10},
			expectedPrev:  "/list?limit=10&page=8",
			expectedNext:  "",
		},
		{
			name:    "Current < 1 hoặc Total < 1",
			current: 0, total: 0, window: 5,
			expectedPages: []int{1}, // Default về current=1, total=1
			expectedPrev:  "",
			expectedNext:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := pagination.NewView(c, tt.current, tt.total, nil, tt.window)

			if view.PrevURL != tt.expectedPrev {
				t.Errorf("PrevURL got %q, want %q", view.PrevURL, tt.expectedPrev)
			}

			if len(view.Pages) != len(tt.expectedPages) {
				t.Fatalf("Pages length got %d, want %d", len(view.Pages), len(tt.expectedPages))
			}
			for i, expectedNum := range tt.expectedPages {
				if view.Pages[i].Num != expectedNum {
					t.Errorf("Pages[%d] Num got %d, want %d", i, view.Pages[i].Num, expectedNum)
				}
			}
		})
	}
}
