package utils

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"social-net/db"
)

func SearchUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "http://social-net.duckdns.org")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	search := strings.TrimSpace(r.URL.Query().Get("search"))
	groupID := r.URL.Query().Get("group_id")

	query := `
		SELECT DISTINCT u.id, u.username, u.email, u.avatar
		FROM users u
		WHERE 1=1
	`
	args := []interface{}{}
	whereClause := ""

	if search != "" {
		whereClause += " AND (LOWER(u.username) LIKE LOWER($1) OR LOWER(u.email) LIKE LOWER($1))"
		args = append(args, "%"+search+"%")
	}

	if groupID != "" {
		whereClause += ` AND u.id NOT IN (
			SELECT user_id 
			FROM group_members 
			WHERE group_id = $` + strconv.Itoa(len(args)+1) + `
		)`
		args = append(args, groupID)
	}

	query += whereClause + " ORDER BY u.username"

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to search users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type User struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Avatar   string `json:"avatar"`
	}

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.Avatar)
		if err != nil {
			http.Error(w, "Failed to scan user", http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}
