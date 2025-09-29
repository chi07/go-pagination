package pagination

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type Paginator struct {
	PerPage     int64
	CurrentPage int64
	TotalItems  int64

	TotalPages  int64
	Offset      int64
	ItemCount   int64
	HasPrevious bool
	HasNext     bool
	PrevPage    int64
	NextPage    int64
}

func NewPaginator(totalItems, currentPage, limit int64) *Paginator {
	p := &Paginator{TotalItems: totalItems, CurrentPage: currentPage, PerPage: limit}
	p.recompute()
	return p
}

func (p *Paginator) recompute() {
	if p.PerPage <= 0 {
		p.PerPage = 10
	}
	if p.TotalItems <= 0 {
		p.TotalItems, p.TotalPages, p.CurrentPage = 0, 1, 1
		p.Offset, p.ItemCount = 0, 0
		p.HasPrevious, p.HasNext = false, false
		p.PrevPage, p.NextPage = 0, 0
		return
	}
	p.TotalPages = (p.TotalItems + p.PerPage - 1) / p.PerPage
	if p.CurrentPage < 1 {
		p.CurrentPage = 1
	} else if p.CurrentPage > p.TotalPages {
		p.CurrentPage = p.TotalPages
	}
	p.Offset = (p.CurrentPage - 1) * p.PerPage
	if p.CurrentPage < p.TotalPages {
		p.ItemCount = p.PerPage
	} else {
		p.ItemCount = p.TotalItems - p.PerPage*(p.TotalPages-1)
	}
	p.HasPrevious, p.HasNext = p.CurrentPage > 1, p.CurrentPage < p.TotalPages
	if p.HasPrevious {
		p.PrevPage = p.CurrentPage - 1
	}
	if p.HasNext {
		p.NextPage = p.CurrentPage + 1
	}
}

type PageItem struct {
	Num    int
	URL    string
	Active bool
}

type View struct {
	Current int
	Total   int
	PrevURL string
	NextURL string
	Pages   []PageItem
}

type URLMode int

const (
	// Relative URL: "/courses?foo=bar&page=2"
	Relative URLMode = iota
	// Absolute URL: "https://example.com/courses?foo=bar&page=2"
	Absolute
)

type BuildOptions struct {
	// Absolute | Relative (mặc định Relative)
	Mode URLMode

	// Path muốn dùng (mặc định: c.Path())
	Path string

	// Tên query param cho trang (mặc định: "page")
	PageParam string

	// Khi Absolute: cách lấy scheme/host (mặc định tự suy ra từ X-Forwarded-Proto/Host, fall back c.Protocol()/c.Hostname())
	Scheme string
	Host   string

	// Có giữ lại các query khác không (mặc định true)
	KeepExistingQuery bool
}

func firstNonEmpty(s ...string) string {
	for _, v := range s {
		if v != "" {
			return v
		}
	}
	return ""
}

func forwardedProto(c *fiber.Ctx) string { return strings.ToLower(c.Get("X-Forwarded-Proto")) }
func forwardedHost(c *fiber.Ctx) string  { return c.Get("X-Forwarded-Host") }

func (o *BuildOptions) normalize(c *fiber.Ctx) {
	// Defaults
	if o.PageParam == "" {
		o.PageParam = "page"
	}
	if o.Path == "" {
		o.Path = c.Path()
	}
	// Mặc định giữ nguyên query nếu caller không set
	if !o.KeepExistingQuery {
		o.KeepExistingQuery = true
	}

	// Absolute URL pieces
	if o.Mode != Absolute {
		return
	}

	// Scheme
	if o.Scheme == "" {
		o.Scheme = firstNonEmpty(forwardedProto(c), c.Protocol())
	}

	// Host
	if o.Host == "" {
		o.Host = firstNonEmpty(forwardedHost(c), c.Hostname())
	}
}

// ------------------ URL Builder ------------------

// BuildPageURL tạo URL cho page cụ thể, giữ lại các query khác (trừ page) nếu chọn KeepExistingQuery.
// BuildPageURL tạo URL cho page cụ thể, giữ lại các query khác (trừ page) nếu chọn KeepExistingQuery.
func BuildPageURL(c *fiber.Ctx, page int, opts *BuildOptions) string {
	var o BuildOptions
	if opts != nil {
		o = *opts
	}
	o.normalize(c)

	q := url.Values{}

	if o.KeepExistingQuery {
		for k, v := range c.Queries() {
			if k == o.PageParam {
				continue
			}
			q.Set(k, v)
		}
	}

	q.Set(o.PageParam, strconv.Itoa(page))

	if o.Mode == Absolute {
		u := &url.URL{
			Scheme:   o.Scheme,
			Host:     o.Host,
			Path:     o.Path,
			RawQuery: q.Encode(),
		}
		return u.String()
	}

	// Relative
	return o.Path + "?" + q.Encode()
}

// NewView dựng PaginationVM cho template.
// window = 0 nghĩa là render hết 1..Total; >0 sẽ render sliding window (ví dụ 5 hiển thị [.. 3 4 5 6 7 ..]).
func NewView(c *fiber.Ctx, current, total int, opts *BuildOptions, window int) View {
	if current < 1 {
		current = 1
	}
	if total < 1 {
		total = 1
	}

	vm := View{Current: current, Total: total}

	if current > 1 {
		vm.PrevURL = BuildPageURL(c, current-1, opts)
	}
	if current < total {
		vm.NextURL = BuildPageURL(c, current+1, opts)
	}

	start, end := 1, total
	if window > 0 && window < total {
		half := window / 2
		start = current - half
		if start < 1 {
			start = 1
		}
		end = start + window - 1
		if end > total {
			end = total
			start = end - window + 1
			if start < 1 {
				start = 1
			}
		}
	}

	vm.Pages = make([]PageItem, 0, end-start+1)
	for i := start; i <= end; i++ {
		vm.Pages = append(vm.Pages, PageItem{
			Num:    i,
			URL:    BuildPageURL(c, i, opts),
			Active: i == current,
		})
	}
	return vm
}
