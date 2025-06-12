package session

import (
	"encoding/json"
	"fmt"
	"net/http"

	"social-net/db"
)

func Middleware(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "http://social-net.duckdns.org")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	re, err := r.Cookie("token")
	fmt.Println("Cookie:", re)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{
			"message": "No token found",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	var sessionCount int
	err = db.DB.QueryRow("SELECT COUNT(*) FROM sessions WHERE token=?", re.Value).Scan(&sessionCount)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Database query failed",
		})
		http.Error(w, "Database query failed", http.StatusInternalServerError)
		return
	}

	if sessionCount > 0 {

		json.NewEncoder(w).Encode(map[string]string{
			"message": "Login successful",
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"message": "Login failed",
	})
}
