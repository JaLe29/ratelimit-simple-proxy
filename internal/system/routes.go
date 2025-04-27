package system

import "net/http"

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RequestURI() == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
