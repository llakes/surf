package jar

import "net/http"

// NewMemoryHeaders creates and readers a new http.Header slice.
func NewMemoryHeaders() http.Header {
	return make(http.Header, 10)
}

// NewJSONHeaders creates and readers a new http.Header slice.
func NewJSONHeaders() http.Header {
	hh := make(http.Header, 10)
	hh.Set("Content-Type", "application/json")
	return hh
}
