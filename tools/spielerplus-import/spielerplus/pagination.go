package spielerplus

import (
	"net/url"
	"strconv"

	"github.com/PuerkitoBio/goquery"
)

// paginationLastPage reads a page's `<ul class="pagination">` (confirmed
// from a HAR capture of a live /cashbox page: page links carry a "page"
// query-string parameter, e.g. "/cashbox/index?page=2&per-page=25") for the
// highest linked page number. Returns 1 if the page has no pagination
// widget at all - confirmed from the same capture to mean "everything fits
// on one page" (e.g. /punishments/index for a club with few enough
// records), not "unknown"/"broken".
func paginationLastPage(doc *goquery.Selection) int {
	last := 1
	doc.Find("ul.pagination a[href]").Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		u, err := url.Parse(href)
		if err != nil {
			return
		}
		page, err := strconv.Atoi(u.Query().Get("page"))
		if err != nil {
			return
		}
		if page > last {
			last = page
		}
	})
	return last
}
