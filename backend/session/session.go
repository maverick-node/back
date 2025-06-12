package session

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"social-net/db"
	logger "social-net/log"

	"github.com/gofrs/uuid"
)

func Setsession(w http.ResponseWriter, r *http.Request, userID string) string {
	Deletesession(userID)

	token, _ := uuid.NewV7()
	sessionID, _ := uuid.NewV7()

	_, err := db.DB.Exec("INSERT INTO sessions (session_id,user_id, token, expires_at) VALUES (?,?, ?, ?)",
		sessionID, userID, token.String(), time.Now().Add(time.Hour*24))
	if err != nil {
		fmt.Println("Error inserting session:", err)
		return ""
	}

	http.SetCookie(w, &http.Cookie{
		Name:    "token",
		Value:   token.String(),
		Expires: time.Now().Add(time.Hour * 2),
		Path:    "/",
	})

	return token.String()
}

func Validatesession(id string, token string) bool {
	var count int
	var expiresAt time.Time

	err := db.DB.QueryRow("SELECT COUNT(*), expires_at FROM sessions WHERE user_id=? AND token=? LIMIT 1",
		id, token).Scan(&count, &expiresAt)
	if err != nil {
		logger.LogError("Error validating session", err)
		return false
	}

	if count == 0 || expiresAt.Before(time.Now()) {
		if count > 0 {
			Deletesession(id)
		}
		return false
	}

	return true
}

func Deletesession(id string) error {
	_, err := db.DB.Exec("DELETE FROM sessions WHERE user_id=?", id)
	if err != nil {
		logger.LogError("Error deleting session", err)
	}
	return nil
}

func Hassession(id string) int {
	var sessionCount int
	err := db.DB.QueryRow("SELECT COUNT(*) FROM sessions WHERE user_id=? AND expires_at > ?",
		id, time.Now()).Scan(&sessionCount)
	if err != nil {
		logger.LogError("Error checking session", err)
		return 0
	}

	return sessionCount
}

func GetUserIDFromToken(token string) (string, bool) {
	if token == "" {
		fmt.Println("Error: Empty token provided")
		return "", false
	}
	var userID string
	err := db.DB.QueryRow("SELECT user_id FROM sessions WHERE token=? AND expires_at > ?",
		token, time.Now()).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("Error: No valid session found for token")
		} else {
			fmt.Println("Database error getting user from token:", err)
		}
		return "", false
	}

	return userID, true
}

func GetUsernameFromUserID(id string) (string, bool) {
	var username string

	err := db.DB.QueryRow("SELECT username FROM users WHERE id=?",
		id).Scan(&username)
	if err != nil {
		logger.LogError("Error getting user from username", err)
		return "", false
	}

	return username, true
}

func GetUserIDFromUsername(username string) (string, error) {
	var userID string
	err := db.DB.QueryRow("SELECT id FROM users WHERE username=? OR email=?",
		username,
		username).Scan(&userID)
	fmt.Println("User ID from username:", userID)
	if err != nil {
		fmt.Println("Error getting user from username:", err)
		return "", err
	}

	return userID, nil
}
