package posts

import (
	"encoding/json"
	"net/http"
	"social-net/db"
)

type PostPrv struct {
	ID       int    `json:"id"`
	PostID   int    `json:"post_id"`
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
}

func PostPrivacy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://frontend-social-so.vercel.app")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	rows, err := db.DB.Query("SELECT * FROM posts_privacy")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var username string
	var post2 []PostPrv
	var posts []PostPrv
	for rows.Next() {
		var post PostPrv
		err := rows.Scan(&post.ID, &post.PostID, &post.UserID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		posts = append(posts, post)
	}
	for i := 0; i < len(posts); i++ {
		db.DB.QueryRow("SELECT username FROM users WHERE id = $1", posts[i].UserID).Scan(&username)
		posts[i].Username = username
		post2 = append(post2, posts[i])
	}

	json.NewEncoder(w).Encode(post2)
}
