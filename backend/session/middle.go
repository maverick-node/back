package session

import (
	"encoding/json"
	"fmt"
	"net/http"

	"social-net/db"
)

func Middleware(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-napk-e1g7awkjb-mavs-projects-a7e88004.vercel.app") // your frontend origin
	w.Header().Set("Access-Control-Allow-Credentials", "true")                                                                // important for cookies
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")                                                      // include all used methods
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")                                                            // accept JSON headers, etc.
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

	// Check if the token exists in the session table
	if sessionCount > 0 {
		// Token is valid, return success
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Login successful",
		})
		return
	}

	// Token not found in the database
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Login failed",
	})
}
