package auth

import (
	"net/http"
)

func Auth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://happy-mushroom-01036131e.6.azurestaticapps.net")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.URL.Path == "/api/auth/login" {
		Login(w, r)
	} else if r.URL.Path == "/api/auth/register" {
		Register(w, r)
	} else if r.URL.Path == "/api/auth/logout" {
		Logout(w, r)
	} else {
		http.Error(w, "Invalid endpoint", http.StatusNotFound)
	}
}
