package auth

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"social-net/db"
	logger "social-net/log"
	"social-net/session"
)

type Info struct {
	ID        string `json:"id"`
	Username  string
	Email     string
	Firstname string
	Lastname  string
	Date      string
	Bio       string
	Password  string
	Avatar    string
}

func Getinfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://happy-mushroom-01036131e.6.azurestaticapps.net")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	cookie, err := r.Cookie("token")
	if err != nil {
		logger.LogError("Unauthorized: Missing token", err)
		http.Error(w, "Unauthorized: Missing token", http.StatusUnauthorized)
		return
	}

	token := cookie.Value
	userID, ok := session.GetUserIDFromToken(token)
	if !ok || userID == "" {
		logger.LogError("Unauthorized: Missing token", err)
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}
	username, ok := session.GetUsernameFromUserID(userID)
	if !ok || username == "" {
		logger.LogError("Unauthorized: Invalid token", err)
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}
	var info Info

	avatar := ""
	err = db.DB.QueryRow("SELECT id, username, email, first_name, last_name, date_of_birth,bio, avatar FROM users WHERE username=?", username).Scan(&info.ID, &info.Username, &info.Email, &info.Firstname, &info.Lastname, &info.Date, &info.Bio, &avatar)
	if err != nil {
		logger.LogError("Error retrieving user information", err)
		if err == sql.ErrNoRows {
			logger.LogError("User not found", err)

			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			logger.LogError("Error retrieving user information", err)

			http.Error(w, "Error retrieving user information", http.StatusInternalServerError)
		}
		return
	}

	if avatar != "" {
		info.Avatar = avatar
	} else {
		info.Avatar = ""
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}
