package messages

import (
	"encoding/json"
	"fmt"
	"net/http"
	"social-net/db"
	"social-net/session"
)

type User struct {
	Username string `json:"username"`
	ID       string `json:"id"`
	Avatar   string `json:"avatar"`
	FullName string `json:"full_name"`
}

func OpenChat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-so.vercel.app") // Allow your frontend's origin
	w.Header().Set("Access-Control-Allow-Credentials", "true")                             // Allow credentials (cookies)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")                   // Allowed methods
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")          // Allowed headers

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	// Get current user
	token, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "No authentication token", http.StatusUnauthorized)
		return
	}

	userID, ok := session.GetUserIDFromToken(token.Value)
	if !ok {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Single query to get all users you can chat with (either following or followers)
	query := `
		SELECT DISTINCT u.username, u.id, u.avatar, u.first_name || ' ' || u.last_name
		FROM users u
		JOIN Followers f ON (f.followed_id = u.id AND f.follower_id = $1) OR (f.follower_id = u.id AND f.followed_id = $2)
		WHERE u.id != $3 AND f.status = 'accepted'
		ORDER BY u.username
	`

	rows, err := db.DB.Query(query, userID, userID, userID)
	if err != nil {
		fmt.Println("Query error:", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var username, id, avatar, fullname string
		if err := rows.Scan(&username, &id, &avatar, &fullname); err != nil {
			fmt.Println("Scan error:", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		// Create a User struct and append to the users slice
		user := User{
			Username: username,
			ID:       id,
			Avatar:   avatar,
			FullName: fullname,
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		fmt.Println("Rows error:", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(users); err != nil {
		fmt.Println("Error encoding response:", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}
