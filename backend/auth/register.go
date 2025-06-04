package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"social-net/db"
	"social-net/session"

	"github.com/gofrs/uuid"
)

func Register(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-net.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if session.IsLoggedIn(r) {
		response := map[string]interface{}{
			"message": "User already logged in",
			"status":  0,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	err := r.ParseMultipartForm(10 << 20) // 10MB max
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	userJson := r.FormValue("user")
	if userJson == "" {
		http.Error(w, "Missing user data", http.StatusBadRequest)
		return
	}
	log.Println("[Register] User JSON:", userJson)
	var user User
	if err := json.Unmarshal([]byte(userJson), &user); err != nil {
		http.Error(w, "Invalid user JSON", http.StatusBadRequest)
		return
	}

	if err := ValidateUser(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	newpss := Hashpwd(user.Password)
	privacy := "public"
	avatarFilename := ""

	// Handle avatar upload
	file, handler, err := r.FormFile("avatar")
	if err == nil && file != nil {
		defer file.Close()
		// Check file Size
		if handler.Size > 2*1024*1024 {
			http.Error(w, "Avatar file too large", http.StatusBadRequest)
			return
		}
		// Check file type
		buff := make([]byte, 512)
		_, _ = file.Read(buff)
		contentType := http.DetectContentType(buff)
		file.Seek(0, io.SeekStart)

		allowedTypes := []string{"image/jpeg", "image/png", "image/gif"}
		valid := false
		for _, t := range allowedTypes {
			if t == contentType {
				valid = true
				break
			}
		}
		if !valid {
			http.Error(w, "Invalid avatar file type", http.StatusBadRequest)
			return
		}
		// Generate new file name: username_timestamp.ext
		ext := filepath.Ext(handler.Filename)
		safeFilename := fmt.Sprintf("%s_%d%s", user.Username, time.Now().Unix(), ext)

		path := "./uploads"
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			os.Mkdir(path, os.ModePerm)
		}
		savePath := filepath.Join(path, safeFilename)

		out, err := os.Create(savePath)
		if err != nil {
			log.Println("Failed to save avatar:", err)
			http.Error(w, "Failed to save avatar", http.StatusInternalServerError)
			return
		}
		defer out.Close()
		_, err = io.Copy(out, file)
		if err != nil {
			log.Println("Failed to write avatar:", err)
			http.Error(w, "Failed to write avatar", http.StatusInternalServerError)
			return
		}
		avatarFilename = safeFilename
	}
	user_id, err := uuid.NewV7()
	if err != nil {
		log.Println("Failed to generate UUID:", err)
		http.Error(w, "Unknown Internal Error, Try again", http.StatusInternalServerError)
		return
	}

	username := generateUSername(user.FirstName, user.LastName)
	_, err = db.DB.Exec(`
		INSERT INTO users (id, username, email, password, first_name, last_name, date_of_birth, bio, privacy, avatar, nickname)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		user_id, username, user.Email, newpss, user.FirstName, user.LastName, user.Birthday, user.Bio, privacy, avatarFilename, user.Nickname)
	if err != nil {
		log.Println("DB error:", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	session.Setsession(w, r, user_id.String())
	log.Println("[Register] Success:", username)
	json.NewEncoder(w).Encode(map[string]string{"message": "register successful"})
}

func ValidateUser(u *User) error {
	// if u.Username == "" || !regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`).MatchString(u.Username) {
	// 	return errors.New("invalid username")
	// }
	if u.Email == "" || !regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`).MatchString(u.Email) {
		return errors.New("invalid email")
	}
	// if len(u.Password) < 8 || !regexp.MustCompile(`[A-Za-z]`).MatchString(u.Password) || !regexp.MustCompile(`\d`).MatchString(u.Password) {
	// 	return errors.New("password must contain at least 8 characters with letters and numbers")
	// }
	if u.FirstName == "" || !regexp.MustCompile(`^[a-zA-Z]{1,30}$`).MatchString(u.FirstName) {
		return errors.New("invalid first name")
	}
	if u.LastName == "" || !regexp.MustCompile(`^[a-zA-Z]{1,30}$`).MatchString(u.LastName) {
		return errors.New("invalid last name")
	}
	if len(u.Birthday) < 4 || len(u.Birthday) > 10 || !regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`).MatchString(u.Birthday) {
		return errors.New("invalid date format")
	}
	year, err := strconv.Atoi(u.Birthday[:4])
	if err != nil || year < 1940 || year > 2007 {
		return errors.New("invalid or out-of-range birth year")
	}
	return nil
}

func generateUSername(firstName, lastName string) string {
	firstInitial := strings.ToLower(string(firstName[0]))
	lowerLast := strings.ToLower(lastName)
	usernaem := firstInitial + lowerLast
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)`
	for i := 1; i < 100; i++ {
		username := fmt.Sprintf("%s%d", usernaem, i)
		var exists bool
		err := db.DB.QueryRow(query, username).Scan(&exists)
		if err != nil {
			log.Println("DB error:", err)
			return ""
		}
		if !exists {
			return username
		}
	}
	return fmt.Sprintf("%s%d", usernaem, time.Now().UnixNano())
}
