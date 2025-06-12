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
	w.Header().Set("Access-Control-Allow-Origin", "https://happy-mushroom-01036131e.6.azurestaticapps.net")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

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

	query := `
		SELECT DISTINCT u.username, u.id, u.avatar, u.first_name || ' ' || u.last_name
		FROM users u
		JOIN Followers f ON (f.followed_id = u.id AND f.follower_id = ?) OR (f.follower_id = u.id AND f.followed_id = ?)
		WHERE u.id != ? AND f.status = 'accepted'
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
