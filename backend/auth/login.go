package auth

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"social-net/db"
	"social-net/session"
)

type User struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	Bio       string `json:"bio"`
	Birthday  string `json:"birthday"`
	Nickname  string `json:"nickname"`
}

func Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-net.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	log.Println("[Login] Headers set:", w.Header())

	if r.Method == http.MethodOptions {
		log.Println("[Login] OPTIONS request received, returning 200 OK")
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method == http.MethodPost {
		log.Println("[Login] POST request received")

		tokene, err := r.Cookie("token")
		if err == nil {
			token := tokene.Value
			log.Println("[Login] Token cookie found:", token)
			_, err1 := session.GetUserIDFromToken(token)
			if err1 {
				log.Println("[Login] User already logged in with token:", token)
				response := map[string]interface{}{
					"message": "User already logged in",
					"status":  0,
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
				return
			}
		} else {
			log.Println("[Login] No token cookie found:", err)
		}

		var user User
		decodeErr := json.NewDecoder(r.Body).Decode(&user)
		if decodeErr != nil {
			log.Println("[Login] Error decoding JSON body:", decodeErr)
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		log.Printf("[Login] User struct decoded: %+v\n", user)

		// LOGIN
		if user.Username != "" && user.Password != "" && user.FirstName == "" {
			log.Println("[Login] Login attempt for user:", user.Username)
			var pass string
			err := db.DB.QueryRow("SELECT password FROM users WHERE username = $1 OR email = $2", user.Username, user.Username).Scan(&pass)
			log.Println("[Login] DB password fetch result:", pass, "err:", err)
			if !Validate(pass, user.Password) {
				log.Println("[Login] Invalid password for user:", user.Username)
				Senddata(w, 2, "Invalid password", "Error Password")
				return
			}
			if err != nil {
				if err == sql.ErrNoRows {
					log.Println("[Login] Username not found:", user.Username)
					http.Error(w, "Invalid username or password", http.StatusUnauthorized)
				} else {
					log.Println("[Login] Database error:", err)
					http.Error(w, "Database error", http.StatusInternalServerError)
				}
				Senddata(w, 0, "Invalid password", "Error Information")
				return
			}

			user_id, err := session.GetUserIDFromUsername(user.Username)
			if err != nil {
				log.Println("[Login] Error fetching user ID for username:", user.Username, "Error:", err)
				http.Error(w, "Error fetching user ID", http.StatusInternalServerError)
				return
			}
			log.Println("[Login] User ID fetched:", user_id)
			session.Setsession(w, r, user_id)
			log.Println("[Login] Session set for user ID:", user_id)
			response := map[string]interface{}{
				"xyz":     user.Username,
				"message": "Login Success",
				"status":  0,
			}
			log.Println("[Login] Login successful for user:", user.Username)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else {
			log.Println("[Login] Invalid input for login:", user)
			http.Error(w, "Invalid input", http.StatusBadRequest)
		}
	}
}
