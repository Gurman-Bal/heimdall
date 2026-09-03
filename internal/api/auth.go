package api

import (
	"net/http"

	"heimdall/internal/auth"
)

func basicAuth(store *auth.Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || !store.Verify(user, pass) {
			w.Header().Set("WWW-Authenticate", `Basic realm="Heimdall"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
