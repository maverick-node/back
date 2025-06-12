package posts

import (
	"encoding/json"
	"fmt"
	"net/http"

	"social-net/db"
	logger "social-net/log"
	"social-net/session"
)

type GetPost struct {
	Id            string
	User_id       string
	Author        string
	Avatar        string
	Content       string
	Title         string
	Image         string
	Creation_date string
	Status        string
}

func Getposts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "http://20.56.138.63:8081")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	tokene, err := r.Cookie("token")
	if err != nil {
		logger.LogError("Missing token", err)
		http.Error(w, "Unauthorized: Missing token", http.StatusUnauthorized)
		return
	}
	token := tokene.Value
	userID, error1 := session.GetUserIDFromToken(token)
	if !error1 || userID == "" {
		http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
		return
	}

	query := `
        SELECT DISTINCT p.id, p.author, p.content, p.title, p.user_id, p.creation_date, p.status, u.avatar, p.Image
        FROM posts p
        LEFT JOIN postsPrivacy pp ON p.id = pp.post_id
        LEFT JOIN Followers f ON p.user_id = f.followed_id
		LEFT JOIN users u ON p.user_id = u.id
        WHERE 
            p.status = 'public' OR
            (p.status = 'semi-private' AND pp.user_id = ?) OR
            (f.follower_id = ? AND f.status = 'accepted') OR
            p.user_id = ?
        ORDER BY p.creation_date DESC
    `

	rows, err := db.DB.Query(query, userID, userID, userID)
	if err != nil {
		logger.LogError("Error fetching posts", err)
		http.Error(w, fmt.Sprintf("Error fetching posts: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var posts []GetPost
	for rows.Next() {
		var post GetPost
		err := rows.Scan(&post.Id, &post.Author, &post.Content, &post.Title, &post.User_id, &post.Creation_date, &post.Status, &post.Avatar, &post.Image)
		if err != nil {
			logger.LogError("Error scanning post", err)
			http.Error(w, fmt.Sprintf("Error scanning post: %v", err), http.StatusInternalServerError)
			return
		}
		allowed := CheckUserPostPermission(userID, post.Id)
		if !allowed {
			logger.LogError("Unauthorized access to post", fmt.Errorf("user %s not allowed to access post %s", userID, post.Id))
			continue
		}

		if post.Image != "" {
			post.Image = "http://20.56.138.63:8080/uploads/" + post.Image
		}
		posts = append(posts, post)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}
