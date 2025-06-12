package comments

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"social-net/db"
	"social-net/posts"
	"social-net/session"

	"github.com/gofrs/uuid"
)

type Comments struct {
	PostId        string
	Comment       string    `json:"comment"`
	Author        string    `json:"author"`
	Avatar        string    `json:"avatar"`
	Image         string    `json:"image"`
	Creation_date time.Time `json:"creation_date"`
}

func AddComments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://white-pebble-0a50c5603.6.azurestaticapps.net")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method == "POST" {

		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			fmt.Println("Failed to parse form:", err)
			return
		}

		postId := r.FormValue("post_id")
		commentText := r.FormValue("comment")

		if postId == "" || commentText == "" {
			http.Error(w, "Missing required fields", http.StatusBadRequest)
			return
		}

		if len(commentText) < 1 {
			http.Error(w, "Comment must be at least 1 character long", http.StatusBadRequest)
			return
		}

		if len(commentText) > 500 {
			http.Error(w, "Comment must not exceed 500 characters", http.StatusBadRequest)
			return
		}

		token, err1 := r.Cookie("token")
		if err1 != nil {
			http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
			return
		}
		tokenn := token.Value
		userid, ok := session.GetUserIDFromToken(tokenn)
		if !ok || userid == "" {
			fmt.Println("Unauthorized: Invalid token:", err)
			http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
			return
		}
		username, _ := session.GetUsernameFromUserID(userid)

		if userid == "" || username == "" || tokenn == "" {
			http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
			return
		}

		allowed := posts.CheckUserPostPermission(userid, postId)
		if !allowed {
			http.Error(w, "Unauthorized: You cannot comment on this post", http.StatusUnauthorized)
			return
		}

		commentID, err := uuid.NewV7()
		if err != nil {
			fmt.Println("Error generating comment ID:", err)
			http.Error(w, "Failed to generate comment ID", http.StatusInternalServerError)
			return
		}

		var comment Comments
		comment.PostId = postId
		comment.Comment = commentText
		comment.Author = username

		file, handler, err := r.FormFile("image")
		if err == nil && file != nil {
			defer file.Close()

			if handler.Size > 2*1024*1024 {
				http.Error(w, "image file too large", http.StatusBadRequest)
				return
			}

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

			ext := filepath.Ext(handler.Filename)
			safeFilename := fmt.Sprintf("%s_%d%s", username, time.Now().Unix(), ext)

			path := "./uploads"
			_, err := os.Stat(path)
			if os.IsNotExist(err) {
				os.MkdirAll(path, os.ModePerm)
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
			comment.Image = safeFilename
			fmt.Println("Image saved successfully:", safeFilename)
		}
		_, err = db.DB.Exec("INSERT INTO comments (id, post_id, author, content,image, creation_date) VALUES (?,?, ?, ?, ?, ?)",
			commentID, comment.PostId, username, comment.Comment, comment.Image, time.Now())
		if err != nil {
			http.Error(w, "Failed to insert comment", http.StatusInternalServerError)
			fmt.Println("Failed to insert comment:", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Comment created successfully",
		})
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

func CheckPostExists(postid string) bool {
	var exists bool
	err := db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM posts WHERE id = ?)", postid).Scan(&exists)
	if err != nil {
		fmt.Println("Error checking if post exists:", err)
		return false
	}
	return exists
}

func Getcomments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://white-pebble-0a50c5603.6.azurestaticapps.net")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	token, _err := r.Cookie("token")
	if _err != nil {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)

		return
	}
	tokenn := token.Value
	userid, ok := session.GetUserIDFromToken(tokenn)
	if !ok || userid == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}
	qu := r.URL.Query()
	postid := qu.Get("post_id")
	fmt.Println("postid", postid)
	if postid == "" {
		http.Error(w, "Missing postid parameter", http.StatusBadRequest)
		return
	}

	if !CheckPostExists(postid) {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}
	allowed := posts.CheckUserPostPermission(userid, postid)
	if !allowed {
		http.Error(w, "Unauthorized: You do not have permission to view this post", http.StatusUnauthorized)
		return
	}
	rows, err := db.DB.Query(`
	SELECT c.post_id, c.content, c.author, u.avatar, c.image, c.creation_date
	FROM comments c
	LEFT JOIN users u ON c.author = u.username
	WHERE c.post_id = ?
	ORDER BY c.creation_date DESC
`, postid)
	if err != nil {
		http.Error(w, "Failed to get comments", http.StatusInternalServerError)
		fmt.Println("Failed to get comments:", err)
		return
	}
	defer rows.Close()
	var comments []Comments
	for rows.Next() {
		var comment Comments
		err := rows.Scan(&comment.PostId, &comment.Comment, &comment.Author, &comment.Avatar, &comment.Image, &comment.Creation_date)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			fmt.Println("Failed to scan comment:", err)
			return
		}
		comments = append(comments, comment)
	}
	json.NewEncoder(w).Encode(comments)
}
