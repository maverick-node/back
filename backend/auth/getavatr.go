package auth

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"social-net/db"
)

func GetAvatar(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "http://20.56.138.63:8081")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		log.Println("[GetAvatar] OPTIONS request received, returning 200 OK")
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method == http.MethodPost {
		log.Println("[GetAvatar] POST request received")
		type Avatar struct {
			Username string `json:"username"`
		}
		var ava Avatar
		json.NewDecoder(r.Body).Decode(&ava)

		query := `
		SELECT avatar FROM users WHERE username = ?
	`
		fmt.Println("username", ava.Username)
		rows, err := db.DB.Query(query, ava.Username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var avatar string
		for rows.Next() {
			err := rows.Scan(&avatar)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		json.NewEncoder(w).Encode(avatar)
	}
}
