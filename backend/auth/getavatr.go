package auth

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"social-net/db"
)

func GetAvatar(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-napk-e1g7awkjb-mavs-projects-a7e88004.vercel.app") // your frontend origin
	w.Header().Set("Access-Control-Allow-Credentials", "true")                                                                // important for cookies
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")                                                      // include all used methods
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")                                                            // accept JSON headers, etc.

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
