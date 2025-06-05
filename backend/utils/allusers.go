package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"social-net/db"
	"social-net/session"
)

type Userss struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Fullname string `json:"fullname"`
	Avatar   string `json:"avatar"`
	Followed bool   `json:"followed"`
}

func Users(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-net.vercel.app") // Allow your frontend's origin
	w.Header().Set("Access-Control-Allow-Credentials", "true")                              // Allow credentials (cookies)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")                    // Allowed methods
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")           // Allowed headers

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	token, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "Unauthorized: Missing token", http.StatusUnauthorized)
		return
	}
	tokenn := token.Value
	user, ok := session.GetUserIDFromToken(tokenn)
	if !ok || user == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}
	username, ok := session.GetUsernameFromUserID(user)
	if !ok {
		http.Error(w, "Failed to get username", http.StatusInternalServerError)
		return
	}

	var users []Userss

	query := `
		SELECT 
			u.id,
			u.username, 
			u.avatar,
			u.first_name || ' ' || u.last_name as fullname,
			CASE 
				WHEN f.status = 'accepted' THEN 1 
				ELSE 0 
			END as followed
		FROM users u
		LEFT JOIN Followers f ON f.followed_id = u.id AND f.follower_id = (SELECT id FROM users WHERE username = $1)
		WHERE u.username != $2`

	rows, err := db.DB.Query(query, username, username)
	if err != nil {
		fmt.Println("Error querying users:", err)
		http.Error(w, "Failed to get users 4", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var user Userss
		var followed int
		err := rows.Scan(&user.ID, &user.Username, &user.Avatar, &user.Fullname, &followed)
		if err != nil {
			fmt.Println("Error scanning user row:", err)
			http.Error(w, "Failed to get users 3", http.StatusInternalServerError)
			return
		}
		user.Followed = followed == 1
		fmt.Println("User fullname:", user.Fullname)
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		fmt.Println("Error iterating through users:", err)
		http.Error(w, "Failed to get users 1", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(users); err != nil {
		fmt.Println("Error encoding users response:", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
