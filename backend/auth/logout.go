package auth

import (
	"encoding/json"
	"net/http"
	"time"

	logger "social-net/log"
	"social-net/session"
)

func Logout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://happy-mushroom-01036131e.6.azurestaticapps.net")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	cookie, err := r.Cookie("token")
	if err != nil {
		logger.LogError("Unauthorized: No token found", err)

		http.Error(w, "Unauthorized - No token found", http.StatusUnauthorized)
		return
	}

	token := cookie.Value

	userid, ok := session.GetUserIDFromToken(token)
	if !ok || userid == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}

	err = session.Deletesession(userid)
	if err != nil {
		logger.LogError("Failed to delete session", err)
		http.Error(w, "Failed to delete session", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:    "token",
		Value:   "",
		Expires: time.Now().Add(-time.Hour),
		Path:    "/",
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Logged out successfully",
	})
}
