package handle

import "net/http"

// extractor extracts a string from a request, returning the string and whether it was found.
type extractor struct {
	tag     string
	extract func(*http.Request, string) (string, bool)
}

func extractPath(r *http.Request, name string) (string, bool) {
	v := r.PathValue(name)
	// note: there is no mechanism to tell whether a path value was absent versus empty,
	// so we treat empty as missing.
	return v, v != ""
}

func extractQuery(r *http.Request, name string) (string, bool) {
	q := r.URL.Query()
	if !q.Has(name) {
		return "", false
	}
	return q.Get(name), true
}

func extractCookie(r *http.Request, name string) (string, bool) {
	c, err := r.Cookie(name)
	if err != nil {
		return "", false
	}
	return c.Value, true
}

func extractHeader(r *http.Request, name string) (string, bool) {
	vals := r.Header.Values(name)
	if len(vals) == 0 {
		return "", false
	}
	return vals[0], true
}

// NewExtractor creates an extractor that reads the given struct tag
// and calls fn to extract a value from the request.
func NewExtractor(tag string, fn func(*http.Request, string) (string, bool)) extractor {
	return extractor{tag: tag, extract: fn}
}

var defaultExtractors = []extractor{
	{"path", extractPath},
	{"query", extractQuery},
	{"cookie", extractCookie},
	{"header", extractHeader},
}
