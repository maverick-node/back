package posts

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"social-net/auth"
	"social-net/db"
	logger "social-net/log"

	"social-net/session"

	"github.com/gofrs/uuid"
)

type Posts struct {
	Author        string `json:"author"`
	Content       string `json:"content"`
	Title         string `json:"title"`
	Image         string `json:"image"`
	Creation_date string `json:"creation_date"`
	Status        string `json:"status"`
	AllowedUsers  string `json:"allowed_users"`
}

func Post(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-net.vercel.app") // Allow frontend origin
	w.Header().Set("Access-Control-Allow-Credentials", "true")                              // Allow cookies
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")                    // Allow methods
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")           // Allow headers

	// Handle OPTIONS preflight request
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Handle POST request
	if r.Method == "POST" {
		tokene, err := r.Cookie("token")
		if err != nil {
			logger.LogError("Error getting token", err)
			http.Error(w, "Unauthorized: Missing token", http.StatusUnauthorized)
			return
		}
		token := tokene.Value
		userid, ok := session.GetUserIDFromToken(token)
		if !ok || userid == "" {
			http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
			return
		}

		// Parse multipart form
		err = r.ParseMultipartForm(10 << 20) // 10 MB max memory
		if err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		var post Posts
		post.Title = r.FormValue("title")
		post.Content = r.FormValue("content")
		post.Status = r.FormValue("status")
		post.AllowedUsers = r.FormValue("allowed_users")
		post.Image = ""

		if post.Title == "" || post.Content == "" {
			logger.LogError("Missing required fields", nil)
			http.Error(w, "Missing required fields", http.StatusBadRequest)
			return
		}

		// Add length validation for title
		if len(post.Title) < 1 {
			logger.LogError("Title too short", nil)
			http.Error(w, "Title must be at least 3 characters long", http.StatusBadRequest)
			return
		}

		if len(post.Title) > 100 {
			logger.LogError("Title too long", nil)
			http.Error(w, "Title must not exceed 100 characters", http.StatusBadRequest)
			return
		}

		// Add length validation for content
		if len(post.Content) < 1 {
			logger.LogError("Content too short", nil)
			http.Error(w, "Content must be at least 10 characters long", http.StatusBadRequest)
			return
		}

		if len(post.Content) > 1000 {
			logger.LogError("Content too long", nil)
			http.Error(w, "Content must not exceed 1000 characters", http.StatusBadRequest)
			return
		}

		author, ok := session.GetUsernameFromUserID(userid)
		if !ok || author == "" {
			http.Error(w, "Unauthorized: User not found", http.StatusUnauthorized)
			return
		}
		var postID string

		// Handle image upload
		file, handler, err := r.FormFile("image")
		if err == nil && file != nil {
			defer file.Close()
			// Check file Size
			if handler.Size > 2*1024*1024 {
				http.Error(w, "image file too large", http.StatusBadRequest)
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
				http.Error(w, "Invalid image file type", http.StatusBadRequest)
				return
			}
			// Generate new file name: username_timestamp.ext
			ext := filepath.Ext(handler.Filename)
			safeFilename := fmt.Sprintf("%s_%d%s", post.Image, time.Now().Unix(), ext)

			path := "./uploads"
			_, err := os.Stat(path)
			if os.IsNotExist(err) {
				os.Mkdir(path, os.ModePerm)
			}
			savePath := filepath.Join(path, safeFilename)

			out, err := os.Create(savePath)
			if err != nil {
				log.Println("Failed to save image:", err)
				http.Error(w, "Failed to save image", http.StatusInternalServerError)
				return
			}
			defer out.Close()
			_, err = io.Copy(out, file)
			if err != nil {
				log.Println("Failed to write image:", err)
				http.Error(w, "Failed to write image", http.StatusInternalServerError)
				return
			}
			post.Image = safeFilename
		}
		if strings.ToLower(post.Status) == "public" || strings.ToLower(post.Status) == "private" {
			// Insert post into the database
			uuidV7, err := uuid.NewV7()
			if err != nil {
				http.Error(w, fmt.Sprintf("Error generating UUID: %v", err), http.StatusInternalServerError)
				return
			}
			postID = uuidV7.String()
			_, err = db.DB.Exec("INSERT INTO posts (id, title, content, user_id, author, creation_date, status,image) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)",
				postID, post.Title, post.Content, userid, author, time.Now(), post.Status, post.Image)
			if err != nil {
				fmt.Println("Error inserting post:", err)
				http.Error(w, fmt.Sprintf("Error inserting post: %v", err), http.StatusInternalServerError)
				return
			}
		}
		if post.Status == "semi-private" {
			var userSlice []string
			fmt.Println("", post.AllowedUsers)
			for _, user := range strings.Split(post.AllowedUsers, ",") {
				sessionid, err := session.GetUserIDFromUsername(strings.TrimSpace(user))
				if err != nil {
					auth.Senddata(w, 2, "User not found", 400)
					return
				}
				if sessionid == "" {
					auth.Senddata(w, 2, "User not found", http.StatusBadRequest)
					return
				}
				userSlice = append(userSlice, sessionid)
			}
			for _, user := range userSlice {
				postID, _ := uuid.NewV7()

				_, err = db.DB.Exec("INSERT INTO posts (id, title, content, user_id, author, creation_date, status,image) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)",
					postID, post.Title, post.Content, userid, author, time.Now(), post.Status, post.Image)
				if err != nil {
					http.Error(w, fmt.Sprintf("Error inserting post: %v", err), http.StatusInternalServerError)
					return
				}
				privacyID, _ := uuid.NewV7()

				_, err = db.DB.Exec("INSERT INTO posts_privacy (id, post_id, user_id) VALUES ($1, $2, $3)",
					privacyID, postID, user)
				if err != nil {
					http.Error(w, fmt.Sprintf("Error inserting post privacy: %v", err), http.StatusInternalServerError)
					return
				}
			}
		}

		// Return success message
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "Post created successfully"})
	}
}
