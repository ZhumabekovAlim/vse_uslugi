package handlers

import "net/http"

// getParam returns a path or query parameter value regardless of whether
// the router stores it with a leading colon or not. It also supports the
// standard net/http PathValue API available in recent Go versions.
func getParam(r *http.Request, name string) string {
	if r == nil {
		return ""
	}

	if val := r.URL.Query().Get(":" + name); val != "" {
		return val
	}

	if val := r.URL.Query().Get(name); val != "" {
		return val
	}

	return r.PathValue(name)
}
