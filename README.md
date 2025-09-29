# go-pagination
[![Go Report Card](https://goreportcard.com/badge/github.com/chi07/pagination)](https://goreportcard.com/report/github.com/chi07/pagination)
[![codecov](https://codecov.io/gh/chi07/pagination/branch/main/graph/badge.svg)](https://codecov.io/gh/chi07/pagination)
[![CI](https://github.com/chi07/pagination/actions/workflows/ci.yml/badge.svg)](https://github.com/chi07/pagination/actions/workflows/ci.yml)


Thư viện Pagination cung cấp một giải pháp mạnh mẽ và linh hoạt để xử lý logic phân trang (paging logic) và tạo URL phân trang (pagination URL) trong các ứng dụng Go, đặc biệt hữu ích cho việc xây dựng API hoặc giao diện web.

Thư viện được thiết kế để tách biệt logic tính toán (tính Offset, TotalPages,...) khỏi logic tạo URL (xử lý X-Forwarded-Headers, bảo toàn query params).


🚀 Cài đặt
Sử dụng lệnh go get để cài đặt thư viện:

Bash

go get github.com/chi07/pagination
✨ Tính năng chính
Paginator Logic (Paginator): Tính toán các thông số cần thiết cho truy vấn cơ sở dữ liệu (Offset, ItemCount) và hiển thị (TotalPages, HasNext, PrevPage).

URL Builder (BuildPageURL): Tạo URL trang kế tiếp/trước đó/tùy chọn một cách thông minh:

Tự động phát hiện và xử lý X-Forwarded-Proto và X-Forwarded-Host (hữu ích cho việc triển khai sau proxy/load balancer).

Tự động bảo toàn các query parameter hiện tại (ngoại trừ tham số page).

Hỗ trợ tạo URL tuyệt đối (Absolute) hoặc tương đối (Relative).

View Model (View): Tạo mô hình dữ liệu (view model) sẵn sàng để render thanh phân trang, bao gồm việc tính toán Sliding Window (hiển thị một cửa sổ trang cố định, ví dụ: [... 3 4 5 6 7 ...]).

📖 Hướng dẫn sử dụng
1. Xử lý Logic Phân trang (Paginator)
   Bạn có thể tạo một Paginator để lấy các thông số cần thiết cho truy vấn DB (như LIMIT và OFFSET).
```go
import "github.com/chi07/pagination"

// Giả sử có 100 mục, người dùng đang ở trang 3, giới hạn 15 mục/trang
totalItems := int64(100)
currentPage := int64(3)
limit := int64(15)

p := pagination.NewPaginator(totalItems, currentPage, limit)

fmt.Printf("Total Pages: %d\n", p.TotalPages) // Kết quả: 7
fmt.Printf("Database Offset: %d\n", p.Offset)   // Kết quả: 30 ((3-1) * 15)
fmt.Printf("Items on this page: %d\n", p.ItemCount) // Kết quả: 15
fmt.Printf("Next Page Number: %d\n", p.NextPage) // Kết quả: 4
```

2. Tạo URL Phân trang (BuildPageURL)
   Sử dụng BuildPageURL để tạo liên kết cho một trang cụ thể.
```go
import "net/http"
// import "github.com/chi07/pagination"

// Tạo request giả lập có sẵn các query params khác
r, _ := http.NewRequest("GET", "http://localhost:8080/api/items?q=go&limit=10", nil)

// Tự động bảo toàn 'q=go' và 'limit=10'
nextPageURL := pagination.BuildPageURL(r, 2, nil) 
// nextPageURL: /api/items?limit=10&page=2&q=go (đã sắp xếp query params)
```

Xử lý URL Tuyệt đối (Absolute URL)
Để tạo URL tuyệt đối và xử lý các proxy header (như X-Forwarded-Proto):

```go
// Giả lập request từ proxy
r.Header.Set("X-Forwarded-Proto", "https")
r.Header.Set("X-Forwarded-Host", "api.example.com")

opts := &pagination.BuildOptions{
    Mode: pagination.Absolute,
}

// Tạo URL tuyệt đối cho trang 4
absoluteURL := pagination.BuildPageURL(r, 4, opts) 
// absoluteURL: https://api.example.com/api/items?limit=10&page=4&q=go
```

3. Tạo View Model (NewView)
   Sử dụng NewView để tạo mô hình dữ liệu sẵn sàng cho template, bao gồm cả tính năng Sliding Window.
```go
// Giả sử đang ở trang 5 trên tổng số 20 trang
current := 5
total := 20
windowSize := 5 // Hiển thị 5 trang liên tiếp

view := pagination.NewView(r, current, total, nil, windowSize)

// view.Pages sẽ chứa:
// [PageItem{Num: 3, ...}, PageItem{Num: 4, ...}, PageItem{Num: 5, Active: true, ...}, PageItem{Num: 6, ...}, PageItem{Num: 7, ...}]
```

render thanh phân trang hoàn chỉnh.
```go
{{if .PrevURL}}
    <a href="{{.PrevURL}}">Previous</a>
{{end}}

{{range .Pages}}
    <a href="{{.URL}}" class="{{if .Active}}active{{end}}">{{.Num}}</a>
{{end}}

{{if .NextURL}}
    <a href="{{.NextURL}}">Next</a>
{{end}}
```