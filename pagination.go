package pagination

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Paginator holds all calculated pagination data for a given set of items.
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

// NewPaginator creates a new Paginator instance and computes all derived fields.
func NewPaginator(totalItems, currentPage, limit int64) Paginator {
	p := Paginator{TotalItems: totalItems, CurrentPage: currentPage, PerPage: limit}
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

// PageItem represents a single page link in the navigation view.
type PageItem struct {
	Num    int
	URL    string
	Active bool
}

// View holds the necessary data structure for rendering pagination links in a template.
type View struct {
	Current int
	Total   int
	PrevURL string
	NextURL string
	Pages   []PageItem
}

// URLMode defines whether the generated URLs should be relative or absolute.
type URLMode int

const (
	// Relative URL: "/courses?foo=bar&page=2"
	Relative URLMode = iota
	// Absolute URL: "https://example.com/courses?foo=bar&page=2"
	Absolute
)

// BuildOptions contains configuration for building pagination URLs.
type BuildOptions struct {
	// Mode determines if the URL is Absolute or Relative (default Relative).
	Mode URLMode

	// Path to use for the URL (default: r.URL.Path).
	Path string

	// PageParam is the query parameter name for the page number (default: "page").
	PageParam string

	// Scheme and Host are used when Mode is Absolute.
	// Defaults are inferred from X-Forwarded-Proto/Host or r.URL/r.Host.
	Scheme string
	Host   string

	// KeepExistingQuery determines whether to retain other query parameters (default true).
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

func forwardedProto(r *http.Request) string {
	return strings.ToLower(r.Header.Get("X-Forwarded-Proto"))
}

func forwardedHost(r *http.Request) string { return r.Header.Get("X-Forwarded-Host") }

func (o *BuildOptions) normalize(r *http.Request) {
	// Defaults
	if o.PageParam == "" {
		o.PageParam = "page"
	}
	if o.Path == "" {
		o.Path = r.URL.Path
	}
	if !o.KeepExistingQuery {
		o.KeepExistingQuery = true
	}

	// Absolute URL pieces
	if o.Mode != Absolute {
		return
	}

	// Scheme
	if o.Scheme == "" {
		// Scheme: kiểm tra TLS (nếu request được handle bởi HTTPS server) hoặc X-Forwarded-Proto
		scheme := "http"
		if r.TLS != nil || forwardedProto(r) == "https" {
			scheme = "https"
		}
		o.Scheme = scheme
	}

	// Host
	if o.Host == "" {
		// Host: kiểm tra X-Forwarded-Host rồi đến r.Host
		o.Host = firstNonEmpty(forwardedHost(r), r.Host)
	}
}

// BuildPageURL creates a URL string for a specific page number,
// while optionally preserving other query parameters.
func BuildPageURL(r *http.Request, page int, opts *BuildOptions) string {
	var o BuildOptions
	if opts != nil {
		o = *opts
	}
	o.normalize(r)

	q := url.Values{}

	existingQuery := r.URL.Query()

	if o.KeepExistingQuery {
		for k, v := range existingQuery {
			if k == o.PageParam {
				continue
			}
			// Lấy giá trị đầu tiên cho query (giống hành vi của Fiber/Query thông thường)
			q.Set(k, v[0])
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

// NewView builds the View model for template rendering.
// If window > 0, it renders a sliding window of pages (e.g., [.. 3 4 5 6 7 ..]).
func NewView(r *http.Request, current, total int, opts *BuildOptions, window int) View {
	if current < 1 {
		current = 1
	}
	if total < 1 {
		total = 1
	}

	vm := View{Current: current, Total: total}

	if current > 1 {
		vm.PrevURL = BuildPageURL(r, current-1, opts)
	}
	if current < total {
		vm.NextURL = BuildPageURL(r, current+1, opts)
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
			URL:    BuildPageURL(r, i, opts),
			Active: i == current,
		})
	}
	return vm
}
