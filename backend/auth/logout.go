package auth

import (
	"encoding/json"
	"net/http"
	logger "social-net/log"
	"social-net/session"
	"time"
)

func Logout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-so.vercel.app") // your frontend origin
	w.Header().Set("Access-Control-Allow-Credentials", "true")                             // important for cookies
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")                   // include all used methods
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")                         // accept JSON headers, etc.
	// Retrieve token from cookie
	cookie, err := r.Cookie("token")

	if err != nil {
		logger.LogError("Unauthorized: No token found", err)
		// If cookie is not found or error occurs, return unauthorized error
		http.Error(w, "Unauthorized - No token found", http.StatusUnauthorized)
		return
	}

	token := cookie.Value

	// Get userID from token
	userid, ok := session.GetUserIDFromToken(token)
	if !ok || userid == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}
	// Delete the session
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

	// Send a successful response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Logged out successfully",
	})
}
