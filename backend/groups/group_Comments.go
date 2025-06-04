package groups

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"social-net/db"
	"social-net/session"

	"github.com/gofrs/uuid"
)

type GroupComment struct {
	ID           string    `json:"id"`
	GroupPostID  string    `json:"group_post_id"`
	Author       string    `json:"author"`
	Content      string    `json:"content"`
	Avatar       string    `json:"avatar"`
	Image        string    `json:"image"`
	CreationDate time.Time `json:"creation_date"`
}

// AddGroupComment handles POST /api/groupcomments/add
func AddGroupComment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-napk-e1g7awkjb-mavs-projects-a7e88004.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20) // 10MB max
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		fmt.Println("Failed to parse form:", err)
		return
	}

	postId := r.FormValue("group_post_id")
	commentText := r.FormValue("content")

	if postId == "" || commentText == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Add length validation for comment content
	if len(commentText) < 1 {
		http.Error(w, "Comment must be at least 1 character long", http.StatusBadRequest)
		return
	}

	if len(commentText) > 500 {
		http.Error(w, "Comment must not exceed 500 characters", http.StatusBadRequest)
		return
	}

	token, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}
	userid, ok := session.GetUserIDFromToken(token.Value)
	if !ok || userid == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}
	username, _ := session.GetUsernameFromUserID(userid)
	commentID, err := uuid.NewV7()
	if err != nil {
		http.Error(w, "Failed to generate comment ID", http.StatusInternalServerError)
		return
	}

	// Handle image upload
	var imageFilename string
	file, handler, err := r.FormFile("image")
	if err == nil && handler != nil {
		defer file.Close()
		imageFilename = fmt.Sprintf("%s_%s", commentID.String(), handler.Filename)
		dst, err := os.Create("./uploads/" + imageFilename)
		if err != nil {
			http.Error(w, "Failed to save image", http.StatusInternalServerError)
			return
		}
		defer dst.Close()
		_, err = io.Copy(dst, file)
		if err != nil {
			http.Error(w, "Failed to save image", http.StatusInternalServerError)
			return
		}
	}

	_, err = db.DB.Exec(
		"INSERT INTO group_comments (id, group_post_id, author, content, image, creation_date) VALUES (?, ?, ?, ?, ?, ?)",
		commentID.String(), postId, username, commentText, imageFilename, time.Now(),
	)
	if err != nil {
		http.Error(w, "Failed to insert comment", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Group comment created successfully"})
}

// GetGroupComments handles GET /api/groupcomments?group_post_id=...
func GetGroupComments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-napk-e1g7awkjb-mavs-projects-a7e88004.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	token, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}
	_, ok := session.GetUserIDFromToken(token.Value)
	if !ok {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}
	groupPostID := r.URL.Query().Get("group_post_id")
	if groupPostID == "" {
		http.Error(w, "Missing group_post_id parameter", http.StatusBadRequest)
		return
	}
	rows, err := db.DB.Query(`
        SELECT gc.id, gc.group_post_id, gc.author, u.avatar, gc.content, gc.image, gc.creation_date
        FROM group_comments gc
        LEFT JOIN users u ON gc.author = u.username
        WHERE gc.group_post_id = ?
        ORDER BY gc.creation_date DESC
    `, groupPostID)
	if err != nil {
		http.Error(w, "Failed to get group comments", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var comments []GroupComment
	for rows.Next() {
		var c GroupComment
		err := rows.Scan(&c.ID, &c.GroupPostID, &c.Author, &c.Avatar, &c.Content, &c.Image, &c.CreationDate)
		if err != nil {
			http.Error(w, "Failed to scan comment", http.StatusInternalServerError)
			return
		}
		comments = append(comments, c)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comments)
}
