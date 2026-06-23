// Package pagination provides helpers for handling limit/offset query params.
package pagination

const (
	DefaultLimit = 50
	MaxLimit     = 500
)

// Parse extracts limit and offset from optional pointer params, applying
// defaults (limit=50, offset=0) and capping limit at MaxLimit.
func Parse(limit, offset *int) (l, o int) {
	l, o = DefaultLimit, 0
	if limit != nil && *limit > 0 {
		l = *limit
		if l > MaxLimit {
			l = MaxLimit
		}
	}
	if offset != nil && *offset > 0 {
		o = *offset
	}
	return
}
